package outbox

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrMessageAlreadyExists = errors.New("message already exists")

type Message struct {
	ID          uuid.UUID
	Topic       string
	MessageKey  string
	Payload     []byte
	CreatedAt   time.Time
	PublishedAt *time.Time
}
