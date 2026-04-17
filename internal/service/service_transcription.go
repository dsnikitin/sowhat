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
	"github.com/dsnikitin/sowhat/internal/consts/stage"
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
	Upload(file io.Reader, contentType string) (string, error)
	AsyncRecognize(fileID, mime string) (string, error)
	CheckRecognitionCompleted(taskID string) (string, error)
	DownloadTranscript(fileID string) (string, []string, error)
}

type Summarizer interface {
	Summarize(text string) (string, error)
}

type TranscriptUploader interface {
	Upload(transcript io.Reader, contentType string) (string, error)
}

type Publisher interface {
	SubscribeForEvent(ctx context.Context, meetingID int64, subsriberID uuid.UUID) error
	UnsubscribeFromEvent(meetingID int64, subscriberID uuid.UUID)
	DeleteSubscription(meetingID int64)
	PublishEvent(msg models.TranscriptionCompleteEvent)
}

type TranscriptionRepository interface {
	CreateTranscription(ctx context.Context, meetingID int64) error
	UpdateTranscription(ctx context.Context, tr models.Transcription) error
	UpdateMeeting(ctx context.Context, meeting models.Meeting) error
	GetNotCompletedTranscriptions(ctx context.Context) iter.Seq2[models.Transcription, error]
}

type TranscriptionTxProvider interface {
	DoTx(ctx context.Context, fn func(rTx TranscriptionRepository) error) error
}

type TranscriptionConfig struct {
	WorkersCount          int           `env:"WORKERS_COUNT" yaml:"workers_count"`
	InputQueueLimit       int           `env:"INPUT_QUEUE_LIMIT" yaml:"input_queue_limit"`
	ProcessQueueLimit     int           `env:"PROCESS_QUEUE_LIMIT" yaml:"process_queue_limit"`
	MaxStageAttemptsCount int           `env:"MAX_STAGE_ATTEMPTS_COUNT" yaml:"max_stage_attempts_count"`
	AttemptsInterval      time.Duration `env:"ATTEMPTS_INTERVAL" yaml:"attempts_interval"`
}

func (h TranscriptionConfig) Validate() error {
	return validation.ValidateStruct(&h,
		validation.Field(&h.WorkersCount, validation.Required, validation.Min(1)),
		validation.Field(&h.InputQueueLimit, validation.Required, validation.Min(1)),
		validation.Field(&h.ProcessQueueLimit, validation.Required, validation.Min(1)),
	)
}

type TranscriptionService struct {
	appCtx    context.Context
	cfg       *TranscriptionConfig
	eg        errgroup.Group
	inputCh   chan (*models.Transcription)
	processCh chan (*models.Transcription)
	retryCh   chan (*models.Transcription)
	t         Transcriber
	sum       Summarizer
	tu        TranscriptUploader
	p         Publisher
	r         TranscriptionRepository
	tx        TranscriptionTxProvider
}

func NewTranscriptionService(
	appCtx context.Context, cfg *TranscriptionConfig,
	t Transcriber, sum Summarizer, ch TranscriptUploader, p Publisher,
	r TranscriptionRepository, tx TranscriptionTxProvider,
) *TranscriptionService {
	s := &TranscriptionService{
		appCtx:    appCtx,
		cfg:       cfg,
		inputCh:   make(chan *models.Transcription, cfg.InputQueueLimit),
		processCh: make(chan *models.Transcription, cfg.InputQueueLimit),
		retryCh:   make(chan *models.Transcription, cfg.InputQueueLimit),
		t:         t,
		sum:       sum,
		tu:        ch,
		p:         p,
		r:         r,
		tx:        tx,
	}

	go func() {
		for range s.cfg.WorkersCount {
			s.eg.Go(s.processor)
		}
		s.eg.Wait()
	}()

	return s
}

func (s *TranscriptionService) AsyncTranscribe(ctx context.Context, userID int64, file models.MeetingFile, subscriberID uuid.UUID) error {
	sf := s.t.SupportedFormats()
	if !slices.Contains(sf, format.FromMIME(file.MIME)) {
		return errx.NewUnsupportedAudioFormatError(sf, errors.New("file format is unsupported"))
	}

	minSize, maxSize := s.t.MinAndMaxFileSize()
	if file.Size < minSize || file.Size > maxSize {
		return errx.NewUnsupportedFileSizeError(maxSize, maxSize, errors.New("file size is unsupported"))
	}

	tr := &models.Transcription{
		Meeting: models.Meeting{
			ID:     file.MeetingID,
			UserID: userID,
		},
		FileContent: file.Reader,
		FileMIME:    file.MIME,
	}

	err := s.tx.DoTx(ctx, func(rTx TranscriptionRepository) error {
		if err := rTx.CreateTranscription(ctx, tr.Meeting.ID); err != nil {
			return errors.Wrap(err, "create transcription")
		}

		if err := s.p.SubscribeForEvent(ctx, file.MeetingID, subscriberID); err != nil {
			return errors.Wrap(err, "subscribe for meeting")
		}

		if err := s.putInQueue(tr, s.inputCh); err != nil {
			s.p.UnsubscribeFromEvent(file.MeetingID, subscriberID)
			return errors.Wrap(err, "put to input chan")
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "do tx")
	}

	logger.Log.Infow("Async transcription successfully started", "meeting_id", tr.Meeting.ID)
	return nil
}

func (s *TranscriptionService) RestartNotCompleted() {
	for tr, err := range s.r.GetNotCompletedTranscriptions(s.appCtx) {
		if err != nil {
			logger.Log.Errorw("Failed to get not completed transcriptions", "error", err.Error())
			return
		}

		// TODO - нужно хранить подписки в БД, чтобы восстановить
		// if err := s.p.SubscribeForEvent(tr.Meeting.ID, subscriberID); err != nil {
		// 	logger.Log.Errorw("Failed to subscribe for event",
		// 		"meeting_id", "subscriber_id", "error", err.Error())
		// }

		select {
		case s.processCh <- &tr:
		default:
			logger.Log.Warnw("Failed to restart not completed transcription",
				"meeting_id", tr.Meeting.ID, "error", err.Error())
		}
	}
}

func (s *TranscriptionService) processor() error {
	for {
		select {
		case <-s.appCtx.Done():
			return nil
		case tr := <-s.processCh: // приоритетно берем из основной очереди
			s.process(tr)
		default:
			select {
			case <-s.appCtx.Done():
				return nil
			case tr := <-s.retryCh: // если нет, то из очереди повторов
				if time.Now().After(tr.LastAttemptAt.Add(s.cfg.AttemptsInterval)) {
					s.process(tr)
					continue
				}

				if err := s.putInQueue(tr, s.retryCh); err != nil {
					logger.Log.Warnw("Failed to put in retry queue on too early case", "meeting_id", tr.Meeting.ID)
				}
			default:
				select {
				case <-s.appCtx.Done():
					return nil
				case tr := <-s.inputCh: // новую в последнюю очередь
					s.process(tr)
				}
			}
		}
	}
}

func (s *TranscriptionService) process(tr *models.Transcription) {
	stage := s.defineStage(*tr)
	// Если после выполнения StageFn в PersistFn будет ошибка, то при рестарте будет повтор StageFn,
	// т.к. обновленное состояние tr не сохранится в БД.
	// Вызвать PersistFn в транзакции не получится, потому что до выполнения StageFn нет обновленных данных для PersistFn
	// Если у Transcriber будет АПИ для проверки уже отгруженных файлов или запущенных тасок распознавания,
	// то можно будет использовать его и не повторять этап.
	stageErr := stage.StageFn(tr)
	logResult("Stage job", stage.Name, tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, stageErr)
	if stageErr != nil {
		if s.canRetry(*tr, stageErr) {
			if err := s.putInQueue(tr, s.processCh); err != nil {
				logger.Log.Warnw("Failed to put in process queue for retry after stage error", "meeting_id", tr.Meeting.ID)
			}
			return
		}
		tr.Meeting.IsTranscriptionFailed = true
	}

	persistErr := stage.PersistFn(*tr)
	logResult("Persist", stage.Name, tr.Meeting.ID, tr.Meeting.IsTranscriptionFailed, persistErr)
	if persistErr != nil {
		if err := s.putInQueue(tr, s.processCh); err != nil {
			logger.Log.Warnw("Failed to put in process queue for retry after persist error", "meeting_id", tr.Meeting.ID)
		}
		return
	}

	if !tr.IsCompleted {
		if err := s.putInQueue(tr, s.processCh); err != nil {
			logger.Log.Warnw("Failed to put in process queue for next stage", "meeting_id", tr.Meeting.ID)
		}
	}
}

func (s *TranscriptionService) defineStage(tr models.Transcription) models.TranscriptionStage {
	switch {
	case tr.Meeting.IsTranscriptionFailed:
		return models.NewTranscriptionStage(stage.Finalize, s.finalize, s.updateTranscription)
	case tr.FileContent == nil:
		tr.Meeting.IsTranscriptionFailed = true
		return models.NewTranscriptionStage(stage.Finalize, s.finalize, s.updateTranscription)
	case tr.TranscriberRqFileID == nil:
		return models.NewTranscriptionStage(stage.Upload, s.uploadFileToTranscriber, s.updateTranscription)
	case tr.TranscriberTaskID == nil:
		return models.NewTranscriptionStage(stage.Recognize, s.startRecognition, s.updateTranscription)
	case tr.TranscriberRsFileID == nil:
		return models.NewTranscriptionStage(stage.Poll, s.checkRecognitionIsReady, s.updateTranscription)
	case tr.Meeting.Transcript == nil:
		return models.NewTranscriptionStage(stage.Download, s.downloadTranscript, s.updateMeeting)
	case tr.Meeting.Summary == nil:
		return models.NewTranscriptionStage(stage.Summarize, s.summarize, s.updateMeeting)
	case tr.Meeting.ChatterFileId == nil:
		return models.NewTranscriptionStage(stage.UploadToChatter, s.uploadFileToChatter, s.updateMeeting)
	default:
		return models.NewTranscriptionStage(stage.Finalize, s.finalize, s.updateTranscription)
	}
}

func (s *TranscriptionService) uploadFileToTranscriber(tr *models.Transcription) error {
	fileID, err := s.t.Upload(tr.FileContent, tr.FileMIME)
	if err != nil {
		return errors.Wrap(err, "upload file")
	}
	tr.TranscriberRqFileID = &fileID
	return nil
}

func (s *TranscriptionService) startRecognition(tr *models.Transcription) error {
	taskID, err := s.t.AsyncRecognize(*tr.TranscriberRqFileID, tr.FileMIME)
	if err != nil {
		return errors.Wrap(err, "async recognize")
	}
	tr.TranscriberTaskID = &taskID
	return nil
}

func (s *TranscriptionService) checkRecognitionIsReady(tr *models.Transcription) error {
	fileID, err := s.t.CheckRecognitionCompleted(*tr.TranscriberTaskID)
	if err != nil {
		return errors.Wrap(err, "check recognition task completed")
	}
	tr.TranscriberRsFileID = &fileID
	return nil
}

func (s *TranscriptionService) downloadTranscript(tr *models.Transcription) error {
	rawTranscript, phrases, err := s.t.DownloadTranscript(*tr.TranscriberRsFileID)
	if err != nil {
		return errors.Wrap(err, "download transcript")
	}
	tr.Meeting.RawTranscript = &rawTranscript
	transcript := strings.Join(phrases, ".\n")
	tr.Meeting.Transcript = &transcript
	return nil
}

func (s *TranscriptionService) summarize(tr *models.Transcription) error {
	summary, err := s.sum.Summarize(*tr.Meeting.RawTranscript)
	if err != nil {
		return errors.Wrap(err, "summarize transcript")
	}
	tr.Meeting.Summary = &summary
	return nil
}

func (s *TranscriptionService) uploadFileToChatter(tr *models.Transcription) error {
	fileID, err := s.tu.Upload(strings.NewReader(*tr.Meeting.Transcript), "text/plain")
	if err != nil {
		return errors.Wrap(err, "upload file to chatter")
	}
	tr.Meeting.ChatterFileId = &fileID
	return nil
}

func (s *TranscriptionService) finalize(tr *models.Transcription) error {
	if err := s.updateMeeting(*tr); err != nil {
		return errors.Wrap(err, "update meeting")
	}

	s.notifyComplete(*tr)
	tr.IsCompleted = true
	return nil
}

func (s *TranscriptionService) notifyComplete(tr models.Transcription) {
	msg := models.TranscriptionCompleteEvent{
		MeetingID: tr.Meeting.ID,
		UserID:    tr.Meeting.UserID,
		IsFailed:  tr.Meeting.IsTranscriptionFailed,
	}

	s.p.PublishEvent(msg)
	s.p.DeleteSubscription(tr.Meeting.ID)
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

func (s *TranscriptionService) putInQueue(tr *models.Transcription, queue chan (*models.Transcription)) error {
	attempt := 0
	for {
		select {
		case <-s.appCtx.Done():
			return nil
		case queue <- tr:
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

func (s *TranscriptionService) canRetry(tr models.Transcription, err error) bool {
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

func logResult(action string, stageName stage.Name, meetingID int64, IsFailed bool, err error) {
	switch err {
	case nil:
		logger.Log.Infow(fmt.Sprintf("%s successfully completed", action),
			"stage", stageName, "meeting_id", meetingID)
	default:
		if stageName == stage.Poll {
			if errors.Is(err, errx.ErrRecognitionTaskNotCompleted) {
				return
			}
			if errors.Is(err, errx.ErrRecognitionTaskFailed) {
				logger.Log.Warnw(fmt.Sprintf("%s failed", action),
					"stage", stageName, "meeting_id", meetingID, "isFailed", IsFailed, "error", err.Error())
				return
			}
		}

		logger.Log.Errorw(fmt.Sprintf("%s failed", action),
			"stage", stageName, "meeting_id", meetingID, "isFailed", IsFailed, "error", err.Error())
	}
}
