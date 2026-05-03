package domain

import (
	"fmt"
	"time"
)

type AccountType   string
type NormalBalance string
type EntryType     string

const (
	AccountAsset     AccountType = "asset"
	AccountLiability AccountType = "liability"
	AccountRevenue   AccountType = "revenue"
	AccountExpense   AccountType = "expense"
	AccountEquity    AccountType = "equity"

	NormalDebit  NormalBalance = "debit"
	NormalCredit NormalBalance = "credit"

	EntryDebit  EntryType = "debit"
	EntryCredit EntryType = "credit"
)

type JournalAccount struct {
	ID            string
	BusinessID    string
	Name          string
	AccountType   AccountType
	NormalBalance NormalBalance
	Description   *string
	CreatedAt     time.Time
}

type JournalEntry struct {
	ID          int64
	BusinessID  string
	PaymentID   string
	AccountID   string
	EntryType   EntryType
	AmountCents int64
	Currency    string
	Description string
	ReversalOf  *int64
	CreatedAt   time.Time
}

func (e *JournalEntry) Validate() error {
	if e.BusinessID == "" {
		return fmt.Errorf("business_id is required")
	}
	if e.PaymentID == "" {
		return fmt.Errorf("payment_id is required")
	}
	if e.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	if e.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	if e.Description == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}

// VerifyBalance returns an error if debits != credits.
func VerifyBalance(entries []JournalEntry) error {
	var debits, credits int64
	for _, e := range entries {
		switch e.EntryType {
		case EntryDebit:
			debits += e.AmountCents
		case EntryCredit:
			credits += e.AmountCents
		}
	}
	if debits != credits {
		return fmt.Errorf("journal does not balance: debits=%d credits=%d", debits, credits)
	}
	return nil
}
