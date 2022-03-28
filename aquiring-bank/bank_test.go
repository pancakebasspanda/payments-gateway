package bank

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"payments_gateway/model"
	protos "payments_gateway/protos"
	"strings"
	"testing"
	"time"
)

// Also I would use httpTest package, but I didn't expose the URL when creating a new client, so
// I decided to check the httpTransport layer instead
type RoundTripFunc struct {
	r   func(req *http.Request) *http.Response
	err error
}

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f.r(req), f.err
}

func TestBank_Validate(t *testing.T) {
	ctx := context.Background()

	expected := "dummy data"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, expected)
	}))
	defer svr.Close()

	tests := []struct {
		name      string
		transport RoundTripFunc
		card      model.Card
		expected  bool
		err       error
	}{
		{
			name: "validates cards details and returns true",
			transport: RoundTripFunc{
				r: func(req *http.Request) *http.Response {
					resp := &http.Response{}

					if req.Method != http.MethodPost {
						t.Error("wrong method")
						t.FailNow()
					}

					if req.URL.Host != "0.0.0.0:1080" {
						t.Error("wrong url")
						t.FailNow()
					}

					if req.URL.Path != "/api/v1/validate" {
						t.Error("wrong path")
						t.FailNow()
					}

					resp.StatusCode = 200

					resp.Body = ioutil.NopCloser(strings.NewReader(`{"status":"valid"}`))

					return resp
				},
			},
			card: model.Card{
				Name:     "Bruce",
				Surname:  "Wayne",
				Postcode: "G15 2DN",
				CardType: protos.CardType_VISA.String(),
				CardNum:  "378282246310005",
				Expiry:   "23/4",
				Cvv:      342,
			},
			expected: true,
			err:      nil,
		},
		{
			name: "validate err",
			transport: RoundTripFunc{
				r: func(req *http.Request) *http.Response {
					resp := &http.Response{}

					if req.Method != http.MethodPost {
						t.Error("wrong method")
						t.FailNow()
					}

					if req.URL.Host != "0.0.0.0:1080" {
						t.Error("wrong url")
						t.FailNow()
					}

					if req.URL.Path != "/api/v1/validate" {
						t.Error("wrong path")
						t.FailNow()
					}

					resp.StatusCode = 500

					resp.Body = ioutil.NopCloser(strings.NewReader(`{"error":"valid"}`))

					return resp
				},
			},
			card: model.Card{
				Name:     "Bruce",
				Surname:  "Wayne",
				Postcode: "G15 2DN",
				CardType: protos.CardType_VISA.String(),
				CardNum:  "378282246310005",
				Expiry:   "23/4",
				Cvv:      342,
			},
			expected: false,
			err:      fmt.Errorf("error performing validation request: POST http://0.0.0.0:1080/api/v1/validate giving up after 2 attempt(s)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryClient := retryablehttp.NewClient()
			retryClient.RetryMax = 1
			retryClient.HTTPClient.Timeout = 5 * time.Second
			retryClient.HTTPClient.Transport = tt.transport

			b := Bank{httpClient: retryClient}

			got, err := b.Validate(ctx, tt.card)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBank_Authorize(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		transaction    model.Transaction
		transport      RoundTripFunc
		expectedCode   string
		expectedReason string
		err            error
	}{
		{
			name: "Successfully authorizes the payment",
			transaction: model.Transaction{
				RefID: "",
				Card: model.Card{
					Name:     "Bruce",
					Surname:  "Wayne",
					Postcode: "G15 2DN",
					CardType: protos.CardType_VISA.String(),
					CardNum:  "378282246310005",
					Expiry:   "23/4",
					Cvv:      342,
				},
				Amount:   20.5,
				Currency: "GBP",
			},
			transport: RoundTripFunc{
				r: func(req *http.Request) *http.Response {
					resp := &http.Response{}

					if req.Method != http.MethodPost {
						t.Error("wrong method")
						t.FailNow()
					}

					if req.URL.Host != "0.0.0.0:1080" {
						t.Error("wrong url")
						t.FailNow()
					}

					if req.URL.Path != "/api/v1/authorize" {
						t.Error("wrong path")
						t.FailNow()
					}

					resp.StatusCode = 200

					resp.Body = ioutil.NopCloser(strings.NewReader(`{"code":"00","reason": "approved and completed successfully"}`))

					return resp
				},
			},
			expectedCode:   "00",
			expectedReason: "approved and completed successfully",
			err:            nil,
		},
		{
			name: "Rejects authorizes the payment",
			transaction: model.Transaction{
				RefID: "",
				Card: model.Card{
					Name:     "Bruce",
					Surname:  "Wayne",
					Postcode: "G15 2DN",
					CardType: protos.CardType_VISA.String(),
					CardNum:  "378282246310005",
					Expiry:   "23/4",
					Cvv:      342,
				},
				Amount:   20.5,
				Currency: "GBP",
			},
			transport: RoundTripFunc{
				r: func(req *http.Request) *http.Response {
					resp := &http.Response{}

					if req.Method != http.MethodPost {
						t.Error("wrong method")
						t.FailNow()
					}

					if req.URL.Host != "0.0.0.0:1080" {
						t.Error("wrong url")
						t.FailNow()
					}

					if req.URL.Path != "/api/v1/authorize" {
						t.Error("wrong path")
						t.FailNow()
					}

					resp.StatusCode = 200

					resp.Body = ioutil.NopCloser(strings.NewReader(`{"code":"06","reason": "not enough funds"}`))

					return resp
				},
			},
			expectedCode:   "06",
			expectedReason: "not enough funds",
			err:            nil,
		},
		{
			name: "returns error",
			transaction: model.Transaction{
				RefID: "",
				Card: model.Card{
					Name:     "Bruce",
					Surname:  "Wayne",
					Postcode: "G15 2DN",
					CardType: protos.CardType_VISA.String(),
					CardNum:  "378282246310005",
					Expiry:   "23/4",
					Cvv:      342,
				},
				Amount:   20.5,
				Currency: "GBP",
			},
			transport: RoundTripFunc{
				r: func(req *http.Request) *http.Response {
					resp := &http.Response{}

					if req.Method != http.MethodPost {
						t.Error("wrong method")
						t.FailNow()
					}

					if req.URL.Host != "0.0.0.0:1080" {
						t.Error("wrong url")
						t.FailNow()
					}

					if req.URL.Path != "/api/v1/authorize" {
						t.Error("wrong path")
						t.FailNow()
					}

					resp.StatusCode = 500

					resp.Body = ioutil.NopCloser(strings.NewReader(`{"error":"new error"}`))

					return resp
				},
			},
			err: fmt.Errorf("error performing authorization request: POST http://0.0.0.0:1080/api/v1/authorize giving up after 2 attempt(s)"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryClient := retryablehttp.NewClient()
			retryClient.RetryMax = 1
			retryClient.HTTPClient.Timeout = 5 * time.Second
			retryClient.HTTPClient.Transport = tt.transport

			b := Bank{httpClient: retryClient}

			got, got1, err := b.Authorize(ctx, tt.transaction)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			assert.Equal(t, tt.expectedCode, got)
			assert.Equal(t, tt.expectedReason, got1)

		})
	}
}
