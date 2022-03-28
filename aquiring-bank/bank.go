package bank

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"payments_gateway/model"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	_validateURL  = "http://0.0.0.0:1080/api/v1/validate"
	_authorizeURL = "http://0.0.0.0:1080/api/v1/authorize"
	_submitURL    = "http://0.0.0.0:1080/api/v1/submit"
)

type Client interface {
	Validate(context.Context, model.Card) (bool, error)                      // validate card info
	Authorize(context.Context, model.Transaction) (string, string, error)    // response code(approved denied)
	Submit(context.Context, []*model.Transaction) (map[string]string, error) // submit approved auths for settlement/payments
}

type Bank struct {
	httpClient *retryablehttp.Client
}

// New creates a new client for the acquiring bank service
func New() Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.HTTPClient.Timeout = 5 * time.Second
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		// too many requests
		if resp != nil {
			if resp.StatusCode == http.StatusTooManyRequests {
				return 1 * time.Minute
			}
		}

		// any other error we perform an exponential backoff
		backOff := math.Pow(2, float64(attemptNum)) * float64(min)
		sleep := time.Duration(backOff)
		if float64(sleep) != backOff || sleep > max {
			sleep = max
		}

		return sleep
	}

	return &Bank{
		httpClient: retryClient,
	}
}

// Validate calls the acquiring banks validate endpoint
func (b *Bank) Validate(ctx context.Context, card model.Card) (bool, error) {
	body, err := json.Marshal(card)
	if err != nil {
		return false, err
	}

	req, err := retryablehttp.NewRequest("POST", _validateURL, body)

	if err != nil {
		return false, err
	}

	req = req.WithContext(ctx)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error performing validation request: %s", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		return false, fmt.Errorf("unexpected response status code: %d", resp.StatusCode)
	}

	var status map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {

		return false, err
	}

	if strings.ToLower(status["status"]) == "valid" {
		return true, nil
	}

	if val, ok := status["error"]; ok {
		return false, fmt.Errorf("%s", val)
	}

	return false, nil
}

// Authorize calls the acquiring banks authorize endpoint
func (b *Bank) Authorize(ctx context.Context, transaction model.Transaction) (string, string, error) {
	body, err := json.Marshal(transaction)
	if err != nil {
		return "", "", err
	}

	req, err := retryablehttp.NewRequest("POST", _authorizeURL, body)

	if err != nil {
		return "", "", err
	}

	req = req.WithContext(ctx)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error performing authorization request: %s", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		return "", "", fmt.Errorf("unexpected response status code: %d", resp.StatusCode)
	}

	var status map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {

		return "", "", err
	}

	if code, ok := status["code"]; ok {
		if reason, ok := status["reason"]; ok {
			return code, reason, nil
		}

		return code, "", nil
	}

	return "", "", nil
}

// Submit submits all transactions from the day for payment
func (b *Bank) Submit(ctx context.Context, transaction []*model.Transaction) (map[string]string, error) {
	out := make(map[string]string, 0)

	body, err := json.Marshal(transaction)
	if err != nil {
		return out, err
	}

	req, err := retryablehttp.NewRequest("POST", _submitURL, body)

	if err != nil {
		return out, err
	}

	req = req.WithContext(ctx)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("error performing submit request: %s", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusTooManyRequests {
		return out, fmt.Errorf("unexpected response status code: %d", resp.StatusCode)
	}

	var results []map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {

		return out, err
	}

	for _, res := range results {
		if refID, ok := res["ref_id"]; ok {
			if success, ok := res["success"]; ok {
				if success == "true" {
					continue
				}

			}
			if reason, ok := res["reason"]; ok {
				out[refID] = reason
			}
		}
	}

	return out, nil
}
