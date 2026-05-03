package domain

import (
	"testing"
)

func TestJournalEntry_Validate(t *testing.T) {
	tests := []struct {
		name    string
		e       JournalEntry
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid entry",
			e: JournalEntry{
				BusinessID:  "biz-1",
				PaymentID:   "pay-1",
				AccountID:   "acc-1",
				AmountCents: 500,
				Description: "test entry",
			},
			wantErr: false,
		},
		{
			name: "missing business_id",
			e: JournalEntry{
				PaymentID:   "pay-1",
				AccountID:   "acc-1",
				AmountCents: 500,
				Description: "test entry",
			},
			wantErr: true,
			errMsg:  "business_id is required",
		},
		{
			name: "missing payment_id",
			e: JournalEntry{
				BusinessID:  "biz-1",
				AccountID:   "acc-1",
				AmountCents: 500,
				Description: "test entry",
			},
			wantErr: true,
			errMsg:  "payment_id is required",
		},
		{
			name: "missing account_id",
			e: JournalEntry{
				BusinessID:  "biz-1",
				PaymentID:   "pay-1",
				AmountCents: 500,
				Description: "test entry",
			},
			wantErr: true,
			errMsg:  "account_id is required",
		},
		{
			name: "zero amount",
			e: JournalEntry{
				BusinessID:  "biz-1",
				PaymentID:   "pay-1",
				AccountID:   "acc-1",
				AmountCents: 0,
				Description: "test entry",
			},
			wantErr: true,
			errMsg:  "amount_cents must be positive",
		},
		{
			name: "negative amount",
			e: JournalEntry{
				BusinessID:  "biz-1",
				PaymentID:   "pay-1",
				AccountID:   "acc-1",
				AmountCents: -1,
				Description: "test entry",
			},
			wantErr: true,
			errMsg:  "amount_cents must be positive",
		},
		{
			name: "missing description",
			e: JournalEntry{
				BusinessID:  "biz-1",
				PaymentID:   "pay-1",
				AccountID:   "acc-1",
				AmountCents: 500,
			},
			wantErr: true,
			errMsg:  "description is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.e.Validate()
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

func TestVerifyBalance(t *testing.T) {
	tests := []struct {
		name    string
		entries []JournalEntry
		wantErr bool
	}{
		{
			name:    "empty entries balances",
			entries: []JournalEntry{},
			wantErr: false,
		},
		{
			name: "balanced debit and credit",
			entries: []JournalEntry{
				{EntryType: EntryDebit, AmountCents: 1000},
				{EntryType: EntryCredit, AmountCents: 1000},
			},
			wantErr: false,
		},
		{
			name: "multiple entries balanced",
			entries: []JournalEntry{
				{EntryType: EntryDebit, AmountCents: 600},
				{EntryType: EntryDebit, AmountCents: 400},
				{EntryType: EntryCredit, AmountCents: 1000},
			},
			wantErr: false,
		},
		{
			name: "unbalanced — more debits",
			entries: []JournalEntry{
				{EntryType: EntryDebit, AmountCents: 1500},
				{EntryType: EntryCredit, AmountCents: 1000},
			},
			wantErr: true,
		},
		{
			name: "unbalanced — more credits",
			entries: []JournalEntry{
				{EntryType: EntryDebit, AmountCents: 1000},
				{EntryType: EntryCredit, AmountCents: 1500},
			},
			wantErr: true,
		},
		{
			name: "only debits",
			entries: []JournalEntry{
				{EntryType: EntryDebit, AmountCents: 500},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyBalance(tt.entries)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %q", err.Error())
			}
		})
	}
}
