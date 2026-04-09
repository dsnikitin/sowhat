package service

import (
	"context"
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"
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

type stage string

const (
	upload    stage = "Upload"
	recognize stage = "Recognize"
	poll      stage = "Poll"
	download  stage = "Download"
	summarize stage = "Summarize"
	chat      stage = "Chat"
	finalize  stage = "Finalize"
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

type ChatFilesUploader interface {
	UploadFile(file io.Reader, contentType string) (string, error)
}

type Publisher interface {
	SubscribeForEvent(meetingID int64, subsriberID uuid.UUID) error
	UnsubscribeFromEvent(meetingID int64, subscriberID uuid.UUID)
	DeleteSubscription(meetingID int64)
	PublishEvent(msg models.TranscriptionCompletedMsg)
}

type TranscriptionRepository interface {
	CreateTranscription(ctx context.Context, meetingID int64) error
	UpdateTranscription(ctx context.Context, tr models.Transcription) error
	UpdateMeeting(ctx context.Context, meeting models.Meeting) error
	GetNotCompletedTranscriptions(ctx context.Context) iter.Seq2[models.Transcription, error]
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

type TranscriptionService struct {
	appCtx         context.Context
	cfg            *TranscriptionConfig
	workers        int
	uploadStage    chan (models.Transcription)
	recognizeStage chan (models.Transcription)
	pollStage      chan (models.Transcription)
	downloadStage  chan (models.Transcription)
	summarizeStage chan (models.Transcription)
	chatStage      chan (models.Transcription)
	finalizeStage  chan (models.Transcription)
	stopCh         chan (struct{})
	eg             errgroup.Group
	t              Transcriber
	sum            Summarizer
	ch             ChatFilesUploader
	p              Publisher
	r              TranscriptionRepository
}

func NewTranscriptionService(
	appCtx context.Context, cfg *TranscriptionConfig,
	tr Transcriber, sum Summarizer, ch ChatFilesUploader, p Publisher,
	r TranscriptionRepository,
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
		chatStage:      make(chan models.Transcription, cfg.QueueLimit),
		finalizeStage:  make(chan models.Transcription, cfg.QueueLimit),
		stopCh:         make(chan struct{}),
		t:              tr,
		sum:            sum,
		ch:             ch,
		p:              p,
		r:              r,
	}

	go s.start()

	return s
}

func (s *TranscriptionService) start() {
	for range s.workers {
		s.eg.Go(func() error {
			return s.process(upload, s.uploadStage, s.recognizeStage, s.uploadFileToTranscriber, s.updateTranscription)
		})
		s.eg.Go(func() error {
			return s.process(recognize, s.recognizeStage, s.pollStage, s.startRecognition, s.updateTranscription)
		})
		s.eg.Go(func() error {
			return s.process(poll, s.pollStage, s.downloadStage, s.checkRecognitionIsReady, s.updateTranscription)
		})
		s.eg.Go(func() error {
			return s.process(download, s.downloadStage, s.summarizeStage, s.downloadTranscript, s.updateMeeting)
		})
		s.eg.Go(func() error { // TODO исправить следущую стадию
			return s.process(summarize, s.summarizeStage, s.finalizeStage, s.summarize, s.updateMeeting)
		})
		s.eg.Go(func() error {
			return s.process(chat, s.chatStage, s.finalizeStage, s.uploadFileToChatter, s.updateMeeting)
		})
		s.eg.Go(func() error {
			return s.process(finalize, s.finalizeStage, nil, s.finalize, s.notifyAndComlete)
		})
	}

	s.eg.Wait()
}

func (s *TranscriptionService) RestartNotCompleted() {
	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*30)
	defer cancel()

	for tr, err := range s.r.GetNotCompletedTranscriptions(ctx) {
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

		// TODO - нужно хранить подписки в БД, чтобы восстановить
		// if err := s.p.SubscribeForEvent(tr.Meeting.ID, subscriberID); err != nil {
		// 	logger.Log.Errorw("Failed to subscribe for event",
		// 		"meeting_id", "subscriber_id", "error", err.Error())
		// }

		switch {
		case tr.Meeting.IsTranscriptionFailed:
			err = s.put(s.finalizeStage, tr)
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
		case tr.Meeting.ChatterFileId == nil:
			err = s.put(s.finalizeStage, tr) // TODO исправить стадию
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

	if err := s.p.SubscribeForEvent(file.MeetingID, subscriberID); err != nil {
		return errors.Wrap(err, "subscribe for meeting")
	}

	if err := s.createTranscription(tr); err != nil {
		s.p.UnsubscribeFromEvent(file.MeetingID, subscriberID)
		return errors.Wrap(err, "create transcription")
	}

	logger.Log.Infow("Transcription task successfully started", "meeting_id", tr.Meeting.ID)
	return nil
}

func (s *TranscriptionService) process(
	name stage,
	curStage, nextStage chan models.Transcription,
	stageFn func(*models.Transcription) error,
	afterStageFn func(models.Transcription) error,
) error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case tr := <-curStage:
			var stageErr error
			var stageField *string
			if tr.LastAttemptAt.Add(s.cfg.AttemptsInterval).After(time.Now()) {
				stageErr = errx.ErrTooEarly
			} else {
				switch name {
				case upload:
					stageField = tr.TranscriberRqFileID
				case recognize:
					stageField = tr.TranscriberTaskID
				case poll:
					stageField = tr.TranscriberRsFileID
				case download:
					stageField = tr.Meeting.Transcript
				case summarize:
					stageField = tr.Meeting.Summary
					// case chat:
					// 	stageField = tr.Meeting.ChatterFileId
				}
				if stageField == nil || (name == finalize && tr.Meeting.IsTranscriptionFailed) {
					stageErr = s.doStage(&tr, func() error {
						return stageFn(&tr)
					})
					logStageResult(string(name), tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
				}
			}

			err := s.persistAndForward(&tr, curStage, nextStage, stageErr, func() error {
				return afterStageFn(tr)
			})
			if !errors.Is(stageErr, errx.ErrTooEarly) || err != nil {
				logFrowardOrRetryResult(string(name), tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, err)
			}
		}
	}
}

func (s *TranscriptionService) uploadFileToTranscriber(tr *models.Transcription) error {
	fileID, err := s.t.UploadFile(tr.FileContent, tr.FileMIME)
	tr.TranscriberRqFileID = &fileID
	return errors.Wrap(err, "upload file")
}

func (s *TranscriptionService) startRecognition(tr *models.Transcription) error {
	taskID, err := s.t.AsyncRecognize(*tr.TranscriberRqFileID, tr.FileMIME)
	tr.TranscriberTaskID = &taskID
	return errors.Wrap(err, "async recognize")
}

func (s *TranscriptionService) checkRecognitionIsReady(tr *models.Transcription) error {
	fileID, err := s.t.CheckTaskCompleted(*tr.TranscriberTaskID)
	tr.TranscriberRsFileID = &fileID
	return errors.Wrap(err, "check recognition task completed")
}

func (s *TranscriptionService) downloadTranscript(tr *models.Transcription) error {
	rawTranscript, phrases, err := s.t.DownloadTranscript(*tr.TranscriberRsFileID)
	tr.Meeting.RawTranscript = &rawTranscript
	transcript := strings.Join(phrases, ".\n")
	tr.Meeting.Transcript = &transcript
	return errors.Wrap(err, "download transcript")
}

func (s *TranscriptionService) summarize(tr *models.Transcription) error {
	summary, err := s.sum.Summarize(*tr.Meeting.RawTranscript)
	tr.Meeting.Summary = &summary
	return errors.Wrap(err, "summarize transcript")
}

func (s *TranscriptionService) uploadFileToChatter(tr *models.Transcription) error {
	fileID, err := s.ch.UploadFile(strings.NewReader(*tr.Meeting.Transcript), "text/plain")
	tr.Meeting.ChatterFileId = &fileID
	return errors.Wrap(err, "upload file to chatter")
}

func (s *TranscriptionService) finalize(tr *models.Transcription) error {
	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
	defer cancel()

	if err := s.r.UpdateMeeting(ctx, tr.Meeting); err != nil {
		return errors.Wrap(err, "update meeting")
	}

	tr.IsCompleted = true
	return nil
}

func (s *TranscriptionService) notifyAndComlete(tr models.Transcription) error {
	msg := models.TranscriptionCompletedMsg{
		MeetingID:             tr.Meeting.ID,
		UserID:                tr.Meeting.UserID,
		IsTranscriptionFailed: tr.Meeting.IsTranscriptionFailed,
	}

	s.p.PublishEvent(msg)
	s.p.DeleteSubscription(tr.Meeting.ID)

	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
	defer cancel()

	tr.IsCompleted = true
	err := s.r.UpdateTranscription(ctx, tr)
	return errors.Wrap(err, "update transcription")
}

func (s *TranscriptionService) updateTranscription(tr models.Transcription) error {
	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
	defer cancel()

	err := s.r.UpdateTranscription(ctx, tr)
	return errors.Wrap(err, "update transcription")
}

func (s *TranscriptionService) updateMeeting(tr models.Transcription) error {
	ctx, cancel := context.WithTimeout(s.appCtx, time.Second*10)
	defer cancel()

	err := s.r.UpdateMeeting(ctx, tr.Meeting)
	return errors.Wrap(err, "update meeting")
}

func (s *TranscriptionService) doStage(tr *models.Transcription, stageFn func() error) error {
	doErr := s.doWork(tr, func() error {
		return errors.Wrap(stageFn(), "stage func")
	})
	if doErr != nil {
		tr.Meeting.IsTranscriptionFailed = !s.canRetry(tr, doErr)
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
	if tr.Meeting.IsTranscriptionFailed && !tr.IsCompleted {
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
		!errors.Is(err, errx.ErrRecognitionTaskFailed) &&
		!errors.Is(err, errx.ErrTooLarge) &&
		!errors.Is(err, errx.ErrInternalServer)
}

func logStageResult(stageName string, meetingID int64, IsFailed bool, err error) {
	if err != nil {
		logger.Log.Errorw(fmt.Sprintf("Failed to do %s stage", stageName),
			"meeting_id", meetingID, "isFailed", IsFailed, "error", err.Error())
	} else {
		logger.Log.Infow(fmt.Sprintf("%s stage successfully completed", stageName), "meeting_id", meetingID)
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
