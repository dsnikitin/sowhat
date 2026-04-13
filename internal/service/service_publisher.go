package service

import (
	"context"
	"sync"

	"github.com/dsnikitin/sowhat/internal/models"
	"github.com/dsnikitin/sowhat/internal/pkg/logger"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Subscriber interface {
	GetID() uuid.UUID
	Notify(msg models.TranscriptionCompleteEvent) error
}

type PublisherService struct {
	mu            sync.RWMutex
	subscribers   map[uuid.UUID]Subscriber
	subscribtions map[int64]map[uuid.UUID]Subscriber
}

func NewPublisher() *PublisherService {
	return &PublisherService{
		subscribers:   make(map[uuid.UUID]Subscriber),
		subscribtions: make(map[int64]map[uuid.UUID]Subscriber),
	}
}

func (p *PublisherService) Subscribe(sub Subscriber) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subscribers[sub.GetID()] = sub
}

func (p *PublisherService) SubscribeForEvent(ctx context.Context, meetingID int64, subsriberID uuid.UUID) error {
	// TODO
	// добавить сохранение подписок в базу и использовать для этого ctx

	p.mu.Lock()
	defer p.mu.Unlock()

	subcriber, ok := p.subscribers[subsriberID]
	if !ok {
		return errors.Errorf("Unknown subscriber %d", subsriberID)
	}

	if _, ok := p.subscribtions[meetingID]; !ok {
		p.subscribtions[meetingID] = map[uuid.UUID]Subscriber{subsriberID: subcriber}
	} else {
		if _, ok := p.subscribtions[meetingID][subsriberID]; !ok {
			p.subscribtions[meetingID][subsriberID] = subcriber
		}
	}

	return nil
}

func (p *PublisherService) UnsubscribeFromEvent(meetingID int64, subscriberID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if subscribers, ok := p.subscribtions[meetingID]; ok {
		delete(subscribers, subscriberID)
	}

	if len(p.subscribtions[meetingID]) == 0 {
		delete(p.subscribtions, meetingID)
	}
}

func (p *PublisherService) DeleteSubscription(meetingID int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.subscribtions, meetingID)
}

func (p *PublisherService) PublishEvent(msg models.TranscriptionCompleteEvent) {
	if _, ok := p.subscribtions[msg.MeetingID]; !ok {
		return
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if subscribers, ok := p.subscribtions[msg.MeetingID]; ok {
		for _, subscriber := range subscribers {
			if err := subscriber.Notify(msg); err != nil {
				logger.Log.Warnw("Failed to notify transcription completed",
					"subscriber_id", subscriber.GetID(), "meeting_id", msg.MeetingID, "is_transcription_failed", msg.IsFailed)
			}
		}
	}
}
