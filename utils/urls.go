package config

var DarajaEndpoints = struct {
	StkPushQuery string
	StkPush      string
	Generate     string
}{
	StkPushQuery: "/mpesa/stkpushquery/v1/query",
	StkPush:      "/mpesa/stkpush/v1/processrequest",
	Generate:     "/oauth/v1/generate?grant_type=client_credentials",
}

const (
	DarajaBaseUrl     = "https://api.safaricom.co.ke" // "https://sandbox.safaricom.co.ke"
	DarajaCallbackUrl = "https://api.mymovies.africa/api/v1/mps/conf/1"
	MkwanjaApiBaseUrl = "http://localhost:8080" // "https://api.mkwanja.appify.co.ke/v1"
)
