package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

type Credentials struct {
	Password         string
	CurrentTimestamp string
}

func GetCredentials() Credentials {
	// Generate timestamp
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, second := now.Clock()

	var CurrentTimestamp = fmt.Sprintf("%04d%02d%02d%02d%02d%02d", year, month, day, hour, minute, second)
	var stringToEncode = os.Getenv("MpesaBusinessShortCode") + os.Getenv("MpesaPassKey") + CurrentTimestamp
	var Password = base64.URLEncoding.EncodeToString([]byte(stringToEncode))
	fmt.Println("P.assword:", Password)

	return Credentials{
		Password:         Password,
		CurrentTimestamp: CurrentTimestamp,
	}
}

func GetMpesaAuthToken(consumerKey string, consumerSecret string) string {
	// Concatenate the consumerKey and consumerSecret
	var authString = consumerKey + ":" + consumerSecret

	if os.Getenv("GO_ENV") == "production" {
		// Encode the authString using base64
		fmt.Println("Auth String:", authString)
		return base64.StdEncoding.EncodeToString([]byte(authString))
	} else {
		return os.Getenv("MpesaTestAuthToken")
	}
}

func GetCurrentTimestamp() string {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, second := now.Clock()

	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d",
		year, month, day, hour, minute, second)
}
