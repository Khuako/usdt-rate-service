package outbox

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepo struct {
	messages  []Message
	listErr   error
	markErr   error
	markedIDs []uuid.UUID
	limit     int
	calls     *[]string
}

func (r *fakeRepo) ListPending(ctx context.Context, limit int) ([]Message, error) {
	*r.calls = append(*r.calls, "list")
	r.limit = limit
	return r.messages, r.listErr
}

func (r *fakeRepo) MarkPublished(ctx context.Context, id uuid.UUID) error {
	*r.calls = append(*r.calls, "mark:"+id.String())
	r.markedIDs = append(r.markedIDs, id)
	return r.markErr
}

type publishCall struct {
	topic   string
	key     string
	payload []byte
}

type fakePub struct {
	published []publishCall
	err       error
	calls     *[]string
}

func (p *fakePub) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	*p.calls = append(*p.calls, "publish:"+key)
	p.published = append(p.published, publishCall{
		topic,
		key,
		payload,
	})
	return p.err
}
func TestNewService_PublishPending(t *testing.T) {
	msgs := []Message{
		{uuid.New(), "topic1", "key1", []byte{1, 2, 3}, time.Now(), nil},
		{uuid.New(), "topic2", "key2", []byte{1, 2, 34}, time.Now(), nil},
	}
	errPublish := errors.New("publish error")
	markError := errors.New("mark published")
	listError := errors.New("list pending error")
	tests := []struct {
		name string

		messages   []Message
		listErr    error
		publishErr error
		markErr    error

		wantErr       error
		wantCalls     []string
		wantPublished []publishCall
		wantMarkedIDs []uuid.UUID
	}{
		{name: "no messages", wantCalls: []string{"list"}},
		{
			name:      "error list pending",
			messages:  msgs,
			listErr:   listError,
			wantErr:   listError,
			wantCalls: []string{"list"},
		},
		{
			name:     "success 2 events",
			messages: msgs,
			wantCalls: []string{
				"list",
				"publish:" + msgs[0].MessageKey,
				"mark:" + msgs[0].ID.String(),
				"publish:" + msgs[1].MessageKey,
				"mark:" + msgs[1].ID.String(),
			},
			wantPublished: []publishCall{{
				topic:   msgs[0].Topic,
				key:     msgs[0].MessageKey,
				payload: msgs[0].Payload,
			}, {
				topic:   msgs[1].Topic,
				key:     msgs[1].MessageKey,
				payload: msgs[1].Payload,
			}},
			wantMarkedIDs: []uuid.UUID{msgs[0].ID, msgs[1].ID},
		},
		{
			name:          "error publish 1st event",
			messages:      msgs,
			publishErr:    errPublish,
			wantErr:       errPublish,
			wantPublished: []publishCall{{topic: msgs[0].Topic, key: msgs[0].MessageKey, payload: msgs[0].Payload}},
			wantCalls: []string{
				"list",
				"publish:" + msgs[0].MessageKey,
			},
		},
		{
			name:          "error markPublished",
			messages:      msgs,
			markErr:       markError,
			wantErr:       markError,
			wantPublished: []publishCall{{topic: msgs[0].Topic, key: msgs[0].MessageKey, payload: msgs[0].Payload}},
			wantMarkedIDs: []uuid.UUID{msgs[0].ID},
			wantCalls: []string{
				"list",
				"publish:" + msgs[0].MessageKey,
				"mark:" + msgs[0].ID.String(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			r := fakeRepo{
				messages: tt.messages,
				listErr:  tt.listErr,
				calls:    &calls,
				markErr:  tt.markErr,
			}
			p := fakePub{
				err:   tt.publishErr,
				calls: &calls,
			}
			service := NewService(&r, &p)
			err := service.PublishPending(t.Context())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("wrong error type, want: %v, got:%v", tt.wantErr, err)
			}
			if !slices.Equal(calls, tt.wantCalls) {
				t.Fatalf("calls are not as expected, got: %v, want: %v", calls, tt.wantCalls)
			}
			if r.limit != 100 {
				t.Errorf("list limit: got %d, want 100", r.limit)
			}
			if len(tt.wantPublished) != len(p.published) {
				t.Fatalf("published are not as expected, got: %v, want: %v", p.published, tt.wantPublished)
			}
			for i, want := range tt.wantPublished {
				got := p.published[i]
				if got.topic != want.topic || got.key != want.key || !bytes.Equal(got.payload, want.payload) {
					t.Errorf("publish[%d]: got %+v, want %+v", i, got, want)
				}
			}
			if !slices.Equal(tt.wantMarkedIDs, r.markedIDs) {
				t.Fatalf("wantMarkedIDs are not as expected, got: %v, want: %v", r.markedIDs, tt.wantMarkedIDs)
			}
		})
	}
}
