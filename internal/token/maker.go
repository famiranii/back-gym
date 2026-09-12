package token

import (
	"time"

	"github.com/google/uuid"
)

type Maker interface {
	CreateToken(phone string, userID uuid.UUID, isAdmin bool, duration time.Duration) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}