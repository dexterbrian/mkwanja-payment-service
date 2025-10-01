package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID                uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID            string     `gorm:"size:100;not null" json:"user_id"`
	PaymentProvider   string     `gorm:"size:50;not null" json:"payment_provider"`
	PaymentMethod     string     `gorm:"size:50;not null" json:"payment_method"`
	PhoneNumber       string     `gorm:"size:15;not null" json:"phone_number"`
	Amount            float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Currency          string     `gorm:"size:3;default:'KES'" json:"currency"`
	Status            string     `gorm:"size:50;not null" json:"status"`
	PaymentReference  string     `gorm:"size:100" json:"payment_reference"`
	CheckoutRequestID string     `gorm:"size:100" json:"checkout_request_id"`
	AccountReference  string     `gorm:"size:100" json:"account_reference"`
	TransactionDesc   string     `json:"transaction_desc"`
	IdempotencyKey    string     `gorm:"size:100;unique" json:"idempotency_key"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at"`
}
