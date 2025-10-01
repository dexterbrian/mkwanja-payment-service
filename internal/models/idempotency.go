package models

import (
	"time"

	"github.com/google/uuid"
)

type IdempotencyKey struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	IdempotencyKey string    `gorm:"size:100;unique;not null" json:"idempotency_key"`
	RequestHash    string    `gorm:"size:64;not null" json:"request_hash"`
	ResponseBody   string    `json:"response_body"`
	StatusCode     int       `json:"status_code"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}
