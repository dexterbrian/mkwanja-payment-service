package queue

const (
	TypeProcessSTKWebhook = "webhook:stk"
	TypeProcessB2CWebhook = "webhook:b2c"
	TypeProcessB2BWebhook = "webhook:b2b"
)

// STKWebhookPayload is the payload for STK webhook tasks.
type STKWebhookPayload struct {
	ConsumerID        string `json:"consumer_id"`
	CheckoutRequestID string `json:"checkout_request_id"`
	RawBody           string `json:"raw_body"`
}

// B2CWebhookPayload is the payload for B2C webhook tasks.
type B2CWebhookPayload struct {
	ConsumerID string `json:"consumer_id"`
	RawBody    string `json:"raw_body"`
}

// B2BWebhookPayload is the payload for B2B webhook tasks.
type B2BWebhookPayload struct {
	ConsumerID string `json:"consumer_id"`
	RawBody    string `json:"raw_body"`
}
