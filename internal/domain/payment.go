package domain

import (
	"fmt"
	"regexp"
	"time"
)

type PaymentProvider  string
type PaymentDirection string
type PaymentType      string
type PaymentStatus    string

const (
	ProviderMpesa  PaymentProvider = "mpesa"
	ProviderStripe PaymentProvider = "stripe"

	DirectionInbound  PaymentDirection = "inbound"
	DirectionOutbound PaymentDirection = "outbound"

	TypeSTKPush PaymentType = "stk_push"
	TypeB2C     PaymentType = "b2c"
	TypeB2B     PaymentType = "b2b"
	TypeC2B     PaymentType = "c2b"

	StatusPending    PaymentStatus = "pending"
	StatusProcessing PaymentStatus = "processing"
	StatusCompleted  PaymentStatus = "completed"
	StatusFailed     PaymentStatus = "failed"
	StatusCancelled  PaymentStatus = "cancelled"
	StatusReversed   PaymentStatus = "reversed"
)

var phoneRegexp = regexp.MustCompile(`^2547\d{8}$`)

type Payment struct {
	ID                string
	BusinessID        string
	IdempotencyKey    string
	Provider          PaymentProvider
	PaymentType       PaymentType
	Direction         PaymentDirection
	Status            PaymentStatus
	AmountCents       int64
	Currency          string
	PhoneNumber       *string
	ReceiverShortcode *string
	Reference         string
	Description       *string
	ProviderRequestID *string
	ProviderTxID      *string
	ProviderReceipt   *string
	CallbackDelivered bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CompletedAt       *time.Time
}

func (p *Payment) Validate() error {
	if p.BusinessID == "" {
		return fmt.Errorf("business_id is required")
	}
	if p.IdempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}
	if p.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	if p.Reference == "" {
		return fmt.Errorf("reference is required")
	}
	if p.PhoneNumber != nil && !phoneRegexp.MatchString(*p.PhoneNumber) {
		return fmt.Errorf("phone_number must be in format 2547XXXXXXXX")
	}
	return nil
}

// NormalisePhone converts common Kenyan phone formats to 2547XXXXXXXX.
func NormalisePhone(phone string) (string, error) {
	switch {
	case regexp.MustCompile(`^0[17]\d{8}$`).MatchString(phone):
		return "254" + phone[1:], nil
	case regexp.MustCompile(`^\+254[17]\d{8}$`).MatchString(phone):
		return phone[1:], nil
	case phoneRegexp.MatchString(phone):
		return phone, nil
	default:
		return "", fmt.Errorf("unrecognised phone format: %s", phone)
	}
}
