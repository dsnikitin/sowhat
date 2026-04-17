package adapter

import (
	"context"

	"github.com/dsnikitin/sowhat/internal/repository"
	"github.com/dsnikitin/sowhat/internal/service"
)

type TranscriptorTxAdapter struct {
	r *repository.TranscriptionRepository
}

func NewTranscriptorTxAdapter(r *repository.TranscriptionRepository) *TranscriptorTxAdapter {
	return &TranscriptorTxAdapter{r: r}
}

func (a *TranscriptorTxAdapter) DoTx(ctx context.Context, fn func(service.TranscriptionRepository) error) error {
	return a.r.DoTx(ctx, func(rTx *repository.TranscriptionRepository) error {
		return fn(rTx)
	})
}
