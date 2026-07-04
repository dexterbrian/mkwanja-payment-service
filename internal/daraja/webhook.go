package daraja

import (
	"strings"
	"unicode"
)

// NormalizePhone normalizes a phone number to the 2547XXXXXXXX format.
func NormalizePhone(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	normalized := b.String()

	if strings.HasPrefix(normalized, "0") {
		normalized = "254" + strings.TrimPrefix(normalized, "0")
	}

	return normalized
}
