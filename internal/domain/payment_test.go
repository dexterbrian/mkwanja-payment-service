package domain

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestPayment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		p       Payment
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid payment",
			p: Payment{
				BusinessID:     "biz-1",
				IdempotencyKey: "idem-1",
				AmountCents:    1000,
				Reference:      "ref-1",
			},
			wantErr: false,
		},
		{
			name: "missing business_id",
			p: Payment{
				IdempotencyKey: "idem-1",
				AmountCents:    1000,
				Reference:      "ref-1",
			},
			wantErr: true,
			errMsg:  "business_id is required",
		},
		{
			name: "missing idempotency_key",
			p: Payment{
				BusinessID:  "biz-1",
				AmountCents: 1000,
				Reference:   "ref-1",
			},
			wantErr: true,
			errMsg:  "idempotency_key is required",
		},
		{
			name: "zero amount",
			p: Payment{
				BusinessID:     "biz-1",
				IdempotencyKey: "idem-1",
				AmountCents:    0,
				Reference:      "ref-1",
			},
			wantErr: true,
			errMsg:  "amount_cents must be positive",
		},
		{
			name: "negative amount",
			p: Payment{
				BusinessID:     "biz-1",
				IdempotencyKey: "idem-1",
				AmountCents:    -100,
				Reference:      "ref-1",
			},
			wantErr: true,
			errMsg:  "amount_cents must be positive",
		},
		{
			name: "missing reference",
			p: Payment{
				BusinessID:     "biz-1",
				IdempotencyKey: "idem-1",
				AmountCents:    1000,
			},
			wantErr: true,
			errMsg:  "reference is required",
		},
		{
			name: "valid phone number",
			p: Payment{
				BusinessID:     "biz-1",
				IdempotencyKey: "idem-1",
				AmountCents:    1000,
				Reference:      "ref-1",
				PhoneNumber:    strPtr("254712345678"),
			},
			wantErr: false,
		},
		{
			name: "invalid phone number",
			p: Payment{
				BusinessID:     "biz-1",
				IdempotencyKey: "idem-1",
				AmountCents:    1000,
				Reference:      "ref-1",
				PhoneNumber:    strPtr("0712345678"),
			},
			wantErr: true,
			errMsg:  "phone_number must be in format 2547XXXXXXXX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.errMsg)
				}
				if err.Error() != tt.errMsg {
					t.Fatalf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}

func TestNormalisePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "07xx format",
			input: "0712345678",
			want:  "254712345678",
		},
		{
			name:  "01xx format",
			input: "0112345678",
			want:  "254112345678",
		},
		{
			name:  "+254 format",
			input: "+254712345678",
			want:  "254712345678",
		},
		{
			name:  "already normalised",
			input: "254712345678",
			want:  "254712345678",
		},
		{
			name:    "unrecognised format",
			input:   "12345",
			wantErr: true,
		},
		{
			name:    "too short",
			input:   "071234",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalisePhone(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalisePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
