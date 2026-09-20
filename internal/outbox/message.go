package outbox

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID          uuid.UUID
	Topic       string
	MessageKey  string
	Payload     []byte
	CreatedAt   time.Time
	PublishedAt *time.Time
}
