package daraja

import (
	"context"
	"fmt"
)

// C2BRegisterURLRequest represents a C2B URL registration request.
type C2BRegisterURLRequest struct {
	ShortCode       string `json:"ShortCode"`
	ResponseType    string `json:"ResponseType"`
	ConfirmationURL string `json:"ConfirmationURL"`
	ValidationURL   string `json:"ValidationURL"`
}

// C2BRegisterURLResponse represents the response from C2B URL registration.
type C2BRegisterURLResponse struct {
	ConversationID       string `json:"ConversationID"`
	OriginatorCoversationID string `json:"OriginatorCoversationID"`
	ResponseCode         string `json:"ResponseCode"`
	ResponseDescription  string `json:"ResponseDescription"`
}

// RegisterC2BURLs registers validation and confirmation URLs for C2B payments.
func (c *Client) RegisterC2BURLs(ctx context.Context, req C2BRegisterURLRequest) (*C2BRegisterURLResponse, error) {
	var resp C2BRegisterURLResponse
	if err := c.postJSON(ctx, "/mpesa/c2b/v1/registerurl", req, &resp); err != nil {
		return nil, fmt.Errorf("c2b register url: %w", err)
	}
	return &resp, nil
}

// C2BValidationBody represents the C2B validation callback body.
type C2BValidationBody struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}

// C2BConfirmationBody represents the C2B confirmation callback body.
type C2BConfirmationBody struct {
	TransactionType   string `json:"TransactionType"`
	TransID           string `json:"TransID"`
	TransTime         string `json:"TransTime"`
	TransAmount       string `json:"TransAmount"`
	BusinessShortCode string `json:"BusinessShortCode"`
	BillRefNumber     string `json:"BillRefNumber"`
	InvoiceNumber     string `json:"InvoiceNumber"`
	OrgAccountBalance string `json:"OrgAccountBalance"`
	ThirdPartyTransID string `json:"ThirdPartyTransID"`
	MSISDN            string `json:"MSISDN"`
	FirstName         string `json:"FirstName"`
	MiddleName        string `json:"MiddleName"`
	LastName          string `json:"LastName"`
}
