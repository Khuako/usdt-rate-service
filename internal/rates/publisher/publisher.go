package publisher

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Publisher struct {
	client *kgo.Client
}

func New(client *kgo.Client) *Publisher {
	return &Publisher{client}
}
func (p *Publisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	recordId := []byte(key)
	record := kgo.Record{
		Key:   recordId,
		Topic: topic,
		Value: payload,
	}
	results := p.client.ProduceSync(ctx, &record)
	if results.FirstErr() != nil {
		return results.FirstErr()
	}
	return nil
}
