package outbox

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	ListPending(ctx context.Context, limit int) ([]Message, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
}

type Publisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}

type Service struct {
	repo Repository
	pub  Publisher
}

func NewService(r Repository, p Publisher) *Service {
	return &Service{
		repo: r,
		pub:  p,
	}
}

func (s *Service) PublishPending(ctx context.Context) error {
	events, err := s.repo.ListPending(ctx, 100)
	if err != nil {
		return err
	}
	for _, e := range events {
		err = s.pub.Publish(ctx, e.Topic, e.MessageKey, e.Payload)
		if err != nil {
			return err
		}
		err = s.repo.MarkPublished(ctx, e.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
