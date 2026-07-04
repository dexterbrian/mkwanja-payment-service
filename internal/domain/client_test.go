package domain

import (
	"testing"
)

func TestClient_Validate(t *testing.T) {
	tests := []struct {
		name    string
		c       Client
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid client",
			c:       Client{ExternalID: "ext-001", Name: "Acme Ltd"},
			wantErr: false,
		},
		{
			name:    "missing external_id",
			c:       Client{Name: "Acme Ltd"},
			wantErr: true,
			errMsg:  "external_id is required",
		},
		{
			name:    "missing name",
			c:       Client{ExternalID: "ext-001"},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "both fields missing",
			c:       Client{},
			wantErr: true,
			errMsg:  "external_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.Validate()
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
