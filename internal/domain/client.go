package domain

import (
	"fmt"
	"time"
)

type Client struct {
	ID         string
	ExternalID string
	Name       string
	Active     bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ClientCredentials struct {
	ID                          string
	ClientID                    string
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

func (c *Client) Validate() error {
	if c.ExternalID == "" {
		return fmt.Errorf("external_id is required")
	}
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
