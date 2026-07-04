package daraja

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+254 712 345 678", "254712345678"},
		{"0712-345-678", "254712345678"},
		{"254712345678", "254712345678"},
		{"+1-555-123-4567", "15551234567"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePhone(tt.input)
			if got != tt.expected {
				t.Fatalf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
