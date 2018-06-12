package fcm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	// PriorityHigh used for high notification priority
	PriorityHigh = "high"

	// PriorityNormal used for normal notification priority
	PriorityNormal = "normal"

	// HeaderRetryAfter HTTP header constant
	HeaderRetryAfter = "Retry-After"

	// ErrorKey readable error caching
	ErrorKey = "error"

	// MethodPOST indicates http post method
	MethodPOST = "POST"
)

var (
	// retryableErrors whether the error is a retryable
	retryableErrors = map[string]bool{
		"Unavailable":         true,
		"InternalServerError": true,
	}

	// FCMServerURL for testing purposes
	FCMServerURL = "https://fcm.googleapis.com/fcm/send"
)

// FCM  stores client with api key to firebase
type FCM struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewFCM creates a new client
func NewFCM(apiKey string) *FCM {
	return &FCM{
		APIKey:     apiKey,
		HTTPClient: &http.Client{},
	}
}

// NewFCMWithClient creates a new client
func NewFCMWithClient(apiKey string, httpClient *http.Client) *FCM {
	return &FCM{
		APIKey:     apiKey,
		HTTPClient: httpClient,
	}
}

// AuthorizationToken returns the APIKey used by the client
func (f *FCM) AuthorizationToken() string {
	return fmt.Sprintf("key=%v", f.APIKey)
}

// Send message to FCM
func (f *FCM) Send(message Message) (Response, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return Response{}, err
	}

	req, err := http.NewRequest(MethodPOST, FCMServerURL, bytes.NewBuffer(data))
	if err != nil {
		return Response{}, err
	}

	req.Header.Set("Authorization", f.AuthorizationToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.HTTPClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Response{}, fmt.Errorf("%d status code", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	response := Response{}
	if err := json.Unmarshal(body, &response); err != nil {
		return response, err
	}

	response.StatusCode = resp.StatusCode
	response.RetryAfter = resp.Header.Get(HeaderRetryAfter)
	if err := f.Failed(&response); err != nil {
		return response, err
	}
	response.Ok = true

	return response, nil
}

// Failed method indicates if the server couldn't process
// the request in time.
func (f *FCM) Failed(response *Response) error {
	for _, response := range response.Results {
		if retryableErrors[response.Error] {
			return fmt.Errorf("Failed %s", response.Error)
		}
	}

	return nil
}
