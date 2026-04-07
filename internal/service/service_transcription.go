package service

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dsnikitin/sowhat/internal/consts/format"
	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/errx"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type Transcriber interface {
	SupportedFormats() []format.Type
	MinAndMaxFileSize() (int64, int64)
	UploadFile(file io.Reader, contentType string) (string, error)
	AsyncRecognize(fileID, mime string) (string, error)
	CheckTaskCompleted(taskID string) (string, error)
	DownloadTranscript(fileID string) (string, []string, error)
}

type Summarizer interface {
	Summarize(text string) (string, error)
}

type TranscriptionRepository interface {
	CreateTranscription(ctx context.Context, meetingID int64) error
	UpdateTranscription(ctx context.Context, tr models.Transcription) error
	UpdateMeeting(ctx context.Context, meeting models.Meeting) error
	GetNotCompletedTranscriptions(ctx context.Context) ([]models.Transcription, error)
}

type Subscriber interface {
	GetID() uuid.UUID
	Notify(msg models.TranscriptionCompletedMsg) error
}

type TranscriptionService struct {
	appCtx         context.Context
	cfg            *TranscriptionConfig
	workers        int
	uploadStage    chan (models.Transcription)
	recognizeStage chan (models.Transcription)
	pollStage      chan (models.Transcription)
	downloadStage  chan (models.Transcription)
	summarizeStage chan (models.Transcription)
	finalizeStage  chan (models.Transcription)
	stopCh         chan (struct{})
	eg             errgroup.Group
	t              Transcriber
	sum            Summarizer
	r              TranscriptionRepository
	mu             sync.RWMutex
	subscribers    map[uuid.UUID]Subscriber
	subscribtions  map[int64]map[uuid.UUID]Subscriber
}

func NewTranscriptionService(
	appCtx context.Context, cfg *TranscriptionConfig, tr Transcriber, sum Summarizer, r TranscriptionRepository,
) *TranscriptionService {
	s := &TranscriptionService{
		cfg:            cfg,
		appCtx:         appCtx,
		workers:        cfg.WorkersCount,
		uploadStage:    make(chan models.Transcription, cfg.QueueLimit),
		recognizeStage: make(chan models.Transcription, cfg.QueueLimit),
		pollStage:      make(chan models.Transcription, cfg.QueueLimit),
		downloadStage:  make(chan models.Transcription, cfg.QueueLimit),
		summarizeStage: make(chan models.Transcription, cfg.QueueLimit),
		finalizeStage:  make(chan models.Transcription, cfg.QueueLimit),
		stopCh:         make(chan struct{}),
		t:              tr,
		sum:            sum,
		r:              r,
		subscribers:    make(map[uuid.UUID]Subscriber),
		subscribtions:  make(map[int64]map[uuid.UUID]Subscriber),
	}

	go s.start()

	return s
}

func (s *TranscriptionService) start() {
	for range s.workers {
		s.eg.Go(s.upload)
		s.eg.Go(s.recognize)
		s.eg.Go(s.poll)
		s.eg.Go(s.download)
		s.eg.Go(s.summarize)
		s.eg.Go(s.finalize)
	}

	s.eg.Wait()
}

type TranscriptionConfig struct {
	WorkersCount          int           `env:"WORKERS_COUNT" yaml:"workers_count"`
	QueueLimit            int           `env:"QUEUE_LIMIT" yaml:"queue_limit"`
	MaxStageAttemptsCount int           `env:"MAX_STAGE_ATTEMPTS_COUNT" yaml:"max_stage_attempts_count"`
	AttemptsInterval      time.Duration `env:"ATTEMPTS_INTERVAL" yaml:"attempts_interval"`
}

func (h TranscriptionConfig) Validate() error {
	return validation.ValidateStruct(&h,
		validation.Field(&h.WorkersCount, validation.Required, validation.Min(1)),
		validation.Field(&h.QueueLimit, validation.Required, validation.Min(1)),
	)
}

func (s *TranscriptionService) RestartNotCompleted() {
	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*30)
	defer cancel()

	trs, err := s.r.GetNotCompletedTranscriptions(ctx)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			logger.Log.Warnw("Failed to get not completed transcriptions", "error", err.Error())
			return
		default:
			logger.Log.Errorw("Failed to get not completed transcriptions", "error", err.Error())
			return
		}
	}

	for _, tr := range trs {
		switch {
		case tr.TranscriberRqFileID == nil:
			tr.Meeting.IsTranscriptionFailed = true
			err = s.put(s.finalizeStage, tr)
		case tr.TranscriberTaskID == nil:
			err = s.put(s.recognizeStage, tr)
		case tr.TranscriberRsFileID == nil:
			err = s.put(s.pollStage, tr)
		case tr.Meeting.Transcript == nil:
			err = s.put(s.downloadStage, tr)
		case tr.Meeting.Summary == nil:
			err = s.put(s.summarizeStage, tr)
		default:
			err = s.put(s.finalizeStage, tr)
		}
		if err != nil {
			logger.Log.Errorw("Failed to restart transcription", "meeting_id", tr.Meeting.ID, "error", err.Error())
		}
	}
}

func (s *TranscriptionService) AsyncTranscribe(userID int64, file models.File, subscriberID uuid.UUID) error {
	sf := s.t.SupportedFormats()
	if !slices.Contains(sf, format.FromMIME(file.MIME)) {
		return errx.NewUnsupportedAudioFormatError(sf, errors.New("file format is unsupported"))
	}

	minSize, maxSize := s.t.MinAndMaxFileSize()
	if file.Size < minSize || file.Size > maxSize {
		return errx.NewUnsupportedFileSizeError(maxSize, maxSize, errors.New("file size is unsupported"))
	}

	tr := models.Transcription{
		Meeting: models.Meeting{
			ID:     file.MeetingID,
			UserID: userID,
		},
		FileContent: file.Reader,
		FileMIME:    file.MIME,
	}

	if err := s.subscribeForMeeting(file.MeetingID, subscriberID); err != nil {
		return errors.Wrap(err, "subscribe for meeting")
	}

	if err := s.createTranscription(tr); err != nil {
		s.unsubscribeFromMeeting(file.MeetingID, subscriberID)
		return errors.Wrap(err, "create transcription")
	}

	logger.Log.Infow("Transcription task successfully started", "meeting_id", tr.Meeting.ID)
	return nil
}

func (s *TranscriptionService) Subscribe(sub Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[sub.GetID()] = sub
}

func (s *TranscriptionService) subscribeForMeeting(meetingID int64, sbrID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	subcriber, ok := s.subscribers[sbrID]
	if !ok {
		return errors.Errorf("Unknown subscriber %d", sbrID)
	}

	if _, ok := s.subscribtions[meetingID]; !ok {
		s.subscribtions[meetingID] = map[uuid.UUID]Subscriber{sbrID: subcriber}
	} else {
		if _, ok := s.subscribtions[meetingID][sbrID]; !ok {
			s.subscribtions[meetingID][sbrID] = subcriber
		}
	}

	return nil
}

func (s *TranscriptionService) unsubscribeFromMeeting(meetingID int64, subscriberID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if subscribers, ok := s.subscribtions[meetingID]; ok {
		delete(subscribers, subscriberID)
	}

	if len(s.subscribtions[meetingID]) == 0 {
		delete(s.subscribtions, meetingID)
	}
}

func (s *TranscriptionService) deleteSubscription(meetingID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscribtions, meetingID)
}

func (s *TranscriptionService) upload() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-s.uploadStage:
			var stageErr error
			if tr.TranscriberRqFileID == nil {
				stageErr = s.doStage(&tr, func() error {
					fileID, err := s.t.UploadFile(tr.FileContent, tr.FileMIME)
					tr.TranscriberRqFileID = &fileID
					return errors.Wrap(err, "upload file")
				})
				logStageResult("Upload", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
			}

			err := s.persistAndForward(&tr, s.uploadStage, s.recognizeStage, stageErr, func() error {
				ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
				defer cancel()

				err := s.r.UpdateTranscription(ctx, tr)
				return errors.Wrap(err, "update transcription")
			})
			logFrowardOrRetryResult("Upload", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
		}
	}
}

func (s *TranscriptionService) recognize() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-s.recognizeStage:
			var stageErr error
			if tr.TranscriberTaskID == nil {
				stageErr = s.doStage(&tr, func() error {
					taskID, err := s.t.AsyncRecognize(*tr.TranscriberRqFileID, tr.FileMIME)
					tr.TranscriberTaskID = &taskID
					return errors.Wrap(err, "async recognize")
				})
				logStageResult("Recognize", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
			}

			err := s.persistAndForward(&tr, s.recognizeStage, s.pollStage, stageErr, func() error {
				ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
				defer cancel()

				err := s.r.UpdateTranscription(ctx, tr)
				return errors.Wrap(err, "update transcription")
			})
			logFrowardOrRetryResult("Recognize", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
		}
	}
}

func (s *TranscriptionService) poll() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-s.pollStage:
			var stageErr error
			if tr.LastAttemptAt.Add(s.cfg.AttemptsInterval).After(time.Now()) {
				stageErr = errx.ErrTooEarly
			} else {
				stageErr = s.doStage(&tr, func() error {
					fileID, err := s.t.CheckTaskCompleted(*tr.TranscriberTaskID)
					tr.TranscriberRsFileID = &fileID
					return errors.Wrap(err, "check recognition task completed")
				})
				switch {
				case stageErr == nil:
					logger.Log.Infow("Poll stage successfully completed",
						"is_succeeded", !tr.Meeting.IsTranscriptionFailed, "meeting_id", tr.Meeting.ID)
				case errors.Is(stageErr, errx.ErrRecognitionTaskNotCompleted):
					tr.StageAttemptsCount = 0
					logger.Log.Infow("Recognition task is processing", "meeting_id", tr.Meeting.ID)
				case errors.Is(stageErr, errx.ErrRecognitionTaskFailed):
					logger.Log.Infow("Recognition task failed", "meeting_id", tr.Meeting.ID)
				default:
					logger.Log.Errorw("Failed to do poll stage",
						"meeting_id", tr.Meeting.ID, "isFailed", tr.Meeting.IsTranscriptionFailed, "error", stageErr.Error())
				}
			}

			err := s.persistAndForward(&tr, s.pollStage, s.downloadStage, stageErr, func() error {
				ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
				defer cancel()

				err := s.r.UpdateTranscription(ctx, tr)
				return errors.Wrap(err, "update transcription")
			})
			if errors.Is(stageErr, errx.ErrTooEarly) {
				continue
			}
			logFrowardOrRetryResult("Poll", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
		}
	}
}

func (s *TranscriptionService) download() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-s.downloadStage:
			var stageErr error
			if tr.Meeting.Transcript == nil {
				stageErr = s.doStage(&tr, func() error {
					rawTranscript, phrases, err := s.t.DownloadTranscript(*tr.TranscriberRsFileID)
					tr.Meeting.RawTranscript = &rawTranscript
					transcript := strings.Join(phrases, ".\n")
					tr.Meeting.Transcript = &transcript
					return errors.Wrap(err, "download transcript")
				})
				logStageResult("Download", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
			}

			err := s.persistAndForward(&tr, s.downloadStage, s.summarizeStage, stageErr, func() error {
				ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
				defer cancel()

				err := s.r.UpdateMeeting(ctx, tr.Meeting)
				return errors.Wrap(err, "update meeting")
			})
			logFrowardOrRetryResult("Download", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
		}
	}
}

func (s *TranscriptionService) summarize() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-s.summarizeStage:
			var stageErr error
			if tr.Meeting.Summary == nil {
				stageErr = s.doStage(&tr, func() error {
					summary, err := s.sum.Summarize(*tr.Meeting.RawTranscript)
					tr.Meeting.Summary = &summary
					return errors.Wrap(err, "summarize transcript")
				})
				logStageResult("Summarize", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
			}

			err := s.persistAndForward(&tr, s.summarizeStage, s.finalizeStage, stageErr, func() error {
				ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
				defer cancel()

				err := s.r.UpdateMeeting(ctx, tr.Meeting)
				return errors.Wrap(err, "update meeting")
			})
			logFrowardOrRetryResult("Summarize", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
		}
	}
}

func (s *TranscriptionService) finalize() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-s.finalizeStage:
			var stageErr error
			if tr.Meeting.IsTranscriptionFailed && !tr.IsCompleted {
				stageErr = s.doStage(&tr, func() error {
					ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
					defer cancel()

					if err := s.r.UpdateMeeting(ctx, tr.Meeting); err != nil {
						return errors.Wrap(err, "update meeting")
					}

					tr.IsCompleted = true
					return nil
				})
				logStageResult("Finalize", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
			}

			err := s.persistAndForward(&tr, s.finalizeStage, nil, stageErr, func() error {
				msg := models.TranscriptionCompletedMsg{
					MeetingID:             tr.Meeting.ID,
					UserID:                tr.Meeting.UserID,
					IsTranscriptionFailed: tr.Meeting.IsTranscriptionFailed,
				}

				s.publishTranscriptionCompletedEvent(msg)
				s.deleteSubscription(tr.Meeting.ID)
				return nil
			})
			logFrowardOrRetryResult("Finalize", tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
		}
	}
}

func (s *TranscriptionService) doStage(tr *models.Transcription, stageFn func() error) error {
	doErr := s.doWork(tr, func() error {
		return errors.Wrap(stageFn(), "stage func")
	})
	if doErr != nil {
		tr.Meeting.IsTranscriptionFailed = s.canRetry(tr, doErr)
		return errors.Wrap(doErr, "do work")
	}

	return nil
}

func (s *TranscriptionService) createTranscription(tr models.Transcription) error {
	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
	defer cancel()

	err := s.doWork(&tr, func() error {
		err := s.r.CreateTranscription(ctx, tr.Meeting.ID)
		return errors.Wrap(err, "create transcription")
	})
	if err != nil {
		return errors.Wrap(err, "do work")
	}

	return errors.Wrap(s.put(s.uploadStage, tr), "put to upload stage")
}

func (s *TranscriptionService) persistAndForward(
	tr *models.Transcription, curStage, nextStage chan (models.Transcription), stageErr error, work func() error,
) error {
	// фейл на этапе
	if tr.Meeting.IsTranscriptionFailed {
		return errors.Wrap(s.put(s.finalizeStage, *tr), "put to finalize stage couse transcription is failed")
	}

	// ошибка на этапе, и еще можно сделать ретрай
	if stageErr != nil {
		if err := s.put(curStage, *tr); err != nil {
			tr.Meeting.IsTranscriptionFailed = true
			return errors.Wrap(s.put(s.finalizeStage, *tr), "put to finalize stage cause can't put to current stage")
		}

		return nil
	}

	// этап прошел успешно
	if doErr := s.doWork(tr, work); doErr != nil {
		if !s.canRetry(tr, doErr) {
			return errors.Wrap(s.put(s.finalizeStage, *tr), "put to finalize stage cause can't retry")
		}

		// не смогли сделать записать в БД, пробуем снова вернуть на текущий этап
		if err := s.put(curStage, *tr); err != nil {
			tr.Meeting.IsTranscriptionFailed = true
			return errors.Wrap(s.put(s.finalizeStage, *tr), "put to finalize stage cause can't put to current stage")
		}
		return errors.Wrap(doErr, "do work can reply")
	}

	// в БД запись обновили, отправляем на следующий этап
	if nextStage != nil {
		if err := s.put(nextStage, *tr); err != nil {
			tr.Meeting.IsTranscriptionFailed = true
			return errors.Wrap(s.put(s.finalizeStage, *tr), "put to finalize stage cause can't put to current stage")
		}
	}

	return nil
}

func (s *TranscriptionService) doWork(tr *models.Transcription, work func() error) error {
	tr.LastAttemptAt = time.Now()
	if err := work(); err != nil {
		tr.StageAttemptsCount++
		return errors.Wrap(err, "work")
	}

	tr.StageAttemptsCount = 0
	return nil
}

func (s *TranscriptionService) put(stage chan (models.Transcription), tr models.Transcription) error {
	attempt := 0
	for {
		select {
		case stage <- tr:
			return nil
		default:
			attempt++
			if attempt < s.cfg.MaxStageAttemptsCount {
				time.Sleep(time.Millisecond * 200)
				continue
			}
			return errx.ErrAllWorkersBusy
		}
	}
}

func (s *TranscriptionService) canRetry(tr *models.Transcription, err error) bool {
	var usErr *errx.ErrUnsupportedFileSize
	var ufErr *errx.ErrUnsupportedAudioFormat

	return !tr.Meeting.IsTranscriptionFailed &&
		(tr.StageAttemptsCount < s.cfg.MaxStageAttemptsCount) &&
		!errors.As(err, &usErr) &&
		!errors.As(err, &ufErr) &&
		!errors.Is(err, errx.ErrRecognitionTaskFailed)
}

func logStageResult(stageName string, meetingID int64, IsFailed bool, err error) {
	if err != nil {
		logger.Log.Errorw(fmt.Sprintf("Failed to do %s stage", stageName),
			"meeting_id", meetingID, "isFailed", IsFailed, "error", err.Error())
	} else {
		logger.Log.Infow(fmt.Sprintf("Upload stage successfully completed", stageName), "meeting_id", meetingID)
	}
}

func logFrowardOrRetryResult(stageName string, meetingID int64, IsFailed bool, err error) {
	if err != nil {
		logger.Log.Errorw(fmt.Sprintf("Failed to persist and forward after %s stage", stageName),
			"meeting_id", meetingID, "isFailed", IsFailed, "error", err.Error())
	} else {
		logger.Log.Infow(fmt.Sprintf("Persist and forward successfully completed after %s stage", stageName),
			"meeting_id", meetingID)
	}
}

func (s *TranscriptionService) publishTranscriptionCompletedEvent(msg models.TranscriptionCompletedMsg) {
	if _, ok := s.subscribtions[msg.MeetingID]; !ok {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if subscribers, ok := s.subscribtions[msg.MeetingID]; ok {
		for _, subscriber := range subscribers {
			if err := subscriber.Notify(msg); err != nil {
				logger.Log.Warnw("Failed to notify transcription completed",
					"subscriber_id", subscriber.GetID(), "meeting_id", msg.MeetingID, "is_transcription_failed", msg.IsTranscriptionFailed)
			}
		}
	}
}
