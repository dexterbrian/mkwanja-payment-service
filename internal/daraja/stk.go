package daraja

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"
)

// STKPushRequest represents an STK push initiation request.
type STKPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushResponse represents the response from initiating an STK push.
type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// STKCallbackMetadataItem is a single item in the callback metadata.
type STKCallbackMetadataItem struct {
	Name  string `json:"Name"`
	Value any    `json:"Value"`
}

// STKCallback represents the inner STK callback payload.
type STKCallback struct {
	MerchantRequestID   string                    `json:"MerchantRequestID"`
	CheckoutRequestID   string                    `json:"CheckoutRequestID"`
	ResultCode          int                       `json:"ResultCode"`
	ResultDesc          string                    `json:"ResultDesc"`
	CallbackMetadata    struct {
		Item []STKCallbackMetadataItem `json:"Item"`
	} `json:"CallbackMetadata"`
}

// STKCallbackBody wraps the Safaricom STK callback body.
type STKCallbackBody struct {
	Body struct {
		StkCallback STKCallback `json:"StkCallback"`
	} `json:"Body"`
}

// InitiateSTKPush triggers an M-PESA STK push request.
func (c *Client) InitiateSTKPush(ctx context.Context, req STKPushRequest) (*STKPushResponse, error) {
	if req.BusinessShortCode == "" {
		req.BusinessShortCode = c.shortcode
	}
	if req.Timestamp == "" {
		req.Timestamp = time.Now().Format("20060102150405")
	}
	if req.Password == "" {
		req.Password = stkPassword(c.shortcode, c.passkey, req.Timestamp)
	}
	if req.TransactionType == "" {
		req.TransactionType = "CustomerPayBillOnline"
	}

	var resp STKPushResponse
	if err := c.postJSON(ctx, "/mpesa/stkpush/v1/processrequest", req, &resp); err != nil {
		return nil, fmt.Errorf("stk push: %w", err)
	}
	return &resp, nil
}

// stkPassword builds the STK push password.
func stkPassword(shortcode, passkey, timestamp string) string {
	plain := shortcode + passkey + timestamp
	return base64.StdEncoding.EncodeToString([]byte(plain))
}
