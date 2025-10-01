package models

import (
	"time"

	"github.com/google/uuid"
)

type Log struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CorrelationID string    `gorm:"size:100" json:"correlation_id"`
	Level         string    `gorm:"size:10;not null" json:"level"`
	Message       string    `gorm:"not null" json:"message"`
	Metadata      string    `gorm:"type:jsonb" json:"metadata"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}
