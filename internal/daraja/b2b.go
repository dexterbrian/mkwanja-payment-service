package daraja

import (
	"context"
	"fmt"
)

// B2BRequest represents a B2B payment request.
type B2BRequest struct {
	Initiator              string `json:"Initiator"`
	SecurityCredential     string `json:"SecurityCredential"`
	CommandID              string `json:"CommandID"`
	SenderIdentifierType   string `json:"SenderIdentifierType"`
	ReceiverIdentifierType string `json:"ReceiverIdentifierType"`
	Amount                 string `json:"Amount"`
	PartyA                 string `json:"PartyA"`
	PartyB                 string `json:"PartyB"`
	AccountReference       string `json:"AccountReference"`
	Remarks                string `json:"Remarks"`
	QueueTimeOutURL        string `json:"QueueTimeOutURL"`
	ResultURL              string `json:"ResultURL"`
}

// B2BResponse represents the response from a B2B payment request.
type B2BResponse struct {
	ConversationID          string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode            string `json:"ResponseCode"`
	ResponseDescription     string `json:"ResponseDescription"`
}

// InitiateB2B sends a B2B payment request.
func (c *Client) InitiateB2B(ctx context.Context, req B2BRequest) (*B2BResponse, error) {
	var resp B2BResponse
	if err := c.postJSON(ctx, "/mpesa/b2b/v1/paymentrequest", req, &resp); err != nil {
		return nil, fmt.Errorf("b2b: %w", err)
	}
	return &resp, nil
}
