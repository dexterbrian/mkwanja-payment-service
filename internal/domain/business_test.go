package domain

import (
	"testing"
)

func TestBusiness_Validate(t *testing.T) {
	tests := []struct {
		name    string
		b       Business
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid business",
			b:       Business{ExternalID: "ext-001", Name: "Acme Ltd"},
			wantErr: false,
		},
		{
			name:    "missing external_id",
			b:       Business{Name: "Acme Ltd"},
			wantErr: true,
			errMsg:  "external_id is required",
		},
		{
			name:    "missing name",
			b:       Business{ExternalID: "ext-001"},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "both fields missing",
			b:       Business{},
			wantErr: true,
			errMsg:  "external_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.b.Validate()
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
