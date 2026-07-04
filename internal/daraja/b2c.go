package daraja

import (
	"context"
	"fmt"
)

// B2CRequest represents a B2C payment request.
type B2CRequest struct {
	OriginatorConversationID string `json:"OriginatorConversationID"`
	InitiatorName            string `json:"InitiatorName"`
	SecurityCredential       string `json:"SecurityCredential"`
	CommandID                string `json:"CommandID"`
	Amount                   string `json:"Amount"`
	PartyA                   string `json:"PartyA"`
	PartyB                   string `json:"PartyB"`
	Remarks                  string `json:"Remarks"`
	QueueTimeOutURL          string `json:"QueueTimeOutURL"`
	ResultURL                string `json:"ResultURL"`
	Occasion                 string `json:"Occasion,omitempty"`
}

// B2CResponse represents the response from a B2C payment request.
type B2CResponse struct {
	ConversationID          string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode            string `json:"ResponseCode"`
	ResponseDescription     string `json:"ResponseDescription"`
}

// InitiateB2C sends a B2C payment request.
func (c *Client) InitiateB2C(ctx context.Context, req B2CRequest) (*B2CResponse, error) {
	var resp B2CResponse
	if err := c.postJSON(ctx, "/mpesa/b2c/v3/paymentrequest", req, &resp); err != nil {
		return nil, fmt.Errorf("b2c: %w", err)
	}
	return &resp, nil
}
