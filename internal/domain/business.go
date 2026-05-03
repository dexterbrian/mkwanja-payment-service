package domain

import (
	"fmt"
	"time"
)

type Business struct {
	ID         string
	ExternalID string
	Name       string
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type BusinessCredentials struct {
	ID                          string
	BusinessID                  string
	Shortcode                   string
	ConsumerKeyEncrypted        string
	ConsumerSecretEncrypted     string
	PasskeyEncrypted            string
	InitiatorName               *string
	SecurityCredentialEncrypted *string
	IsActive                    bool
	CreatedAt                   time.Time
}

// DecryptedCredentials holds plaintext credentials — never log or serialise.
type DecryptedCredentials struct {
	Shortcode          string
	ConsumerKey        string
	ConsumerSecret     string
	Passkey            string
	InitiatorName      string
	SecurityCredential string
}

func (b *Business) Validate() error {
	if b.ExternalID == "" {
		return fmt.Errorf("external_id is required")
	}
	if b.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
