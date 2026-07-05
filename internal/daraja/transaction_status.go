package daraja

import (
	"context"
	"fmt"
)

type TransactionStatusRequest struct {
	Initiator          string `json:"Initiator"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	TransactionID      string `json:"TransactionID"`
	PartyA             string `json:"PartyA"`
	IdentifierType     string `json:"IdentifierType"`
	ResultURL          string `json:"ResultURL"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	Remarks            string `json:"Remarks"`
	Occasion           string `json:"Occasion,omitempty"`
}

type TransactionStatusResponse struct {
	ResponseCode                string `json:"ResponseCode"`
	ResponseDescription         string `json:"ResponseDescription"`
	MerchantRequestID           string `json:"MerchantRequestID"`
	CheckoutRequestID           string `json:"CheckoutRequestID"`
	ResultCode                  string `json:"ResultCode"`
	ResultDesc                  string `json:"ResultDesc"`
	ConversationID              string `json:"ConversationID"`
	OriginatorConversationID    string `json:"OriginatorConversationID"`
	ReceiptNo                   string `json:"ReceiptNo"`
	TransactionAmount           string `json:"TransactionAmount"`
	TransactionDate             string `json:"TransactionDate"`
	TransactionStatus           string `json:"TransactionStatus"`
	ReasonType                  string `json:"ReasonType"`
}

func (c *Client) QueryTransactionStatus(ctx context.Context, req TransactionStatusRequest) (*TransactionStatusResponse, error) {
	var resp TransactionStatusResponse
	if err := c.postJSON(ctx, "/mpesa/transactionstatus/v1/query", req, &resp); err != nil {
		return nil, fmt.Errorf("query transaction status: %w", err)
	}
	return &resp, nil
}
