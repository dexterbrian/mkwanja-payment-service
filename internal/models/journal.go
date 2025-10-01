package models

import (
	"time"

	"github.com/google/uuid"
)

type JournalAccount struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      string    `gorm:"size:100" json:"user_id"`
	AccountName string    `gorm:"size:100;not null" json:"account_name"`
	AccountType string    `gorm:"size:50;not null" json:"account_type"`
	Balance     float64   `gorm:"type:decimal(15,2);default:0.00" json:"balance"`
	Currency    string    `gorm:"size:3;default:'KES'" json:"currency"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type JournalEntry struct {
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TransactionID    uuid.UUID `gorm:"type:uuid" json:"transaction_id"`
	JournalAccountID uuid.UUID `gorm:"type:uuid" json:"journal_account_id"`
	Debit            float64   `gorm:"type:decimal(15,2)" json:"debit"`
	Credit           float64   `gorm:"type:decimal(15,2)" json:"credit"`
	Currency         string    `gorm:"size:3;default:'KES'" json:"currency"`
	Description      string    `json:"description"`
	Reference        string    `gorm:"size:100" json:"reference"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}
