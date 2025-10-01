package models

import (
	"time"

	"github.com/google/uuid"
)

type Payment struct {
	ID               uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PaymentReference string     `gorm:"size:100" json:"payment_reference"`
	TransactionDate  *time.Time `json:"transaction_date"`
	PhoneNumber      string     `gorm:"size:15" json:"phone_number"`
	Amount           float64    `gorm:"type:decimal(15,2)" json:"amount"`
	RawPayload       string     `gorm:"type:jsonb;not null" json:"raw_payload"`
	Processed        bool       `gorm:"default:false" json:"processed"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
}
