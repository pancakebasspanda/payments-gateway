package integration_tests

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	protos "payments_gateway/protos"
	"payments_gateway/storage/postgres"
	"testing"
)

func TestPgxStorage_AddAndRetrievePaymentInfo(t *testing.T) {
	ctx := context.Background()
	pool, err := postgres.CreatePgPool(ctx, "postgres://user1:123@localhost:5433/payments_test", 5, 1)
	if err != nil {
		log.WithError(err).Fatal("creating database pgClient")
	}

	refID := "825ca1787c9d4672991848a5bfbc1057"

	pgClient := postgres.New(pool)

	tests := []struct {
		name     string
		refID    string
		request  *protos.ProcessPaymentRequest
		status   protos.Status
		reason   string
		expected protos.GetPaymentResponse
		err      error
	}{
		{
			name:  "stores payment",
			refID: refID,
			request: &protos.ProcessPaymentRequest{
				BillingDetails: &protos.BillingDetails{
					Name:          "Bruce",
					Surname:       "Wayne",
					Email:         "iam@batman.com",
					Phone:         "0789825678",
					AddressLine_1: "wayne manor",
					AddressLine_2: "Gotham city",
					Postcode:      "G15 2DN",
				},
				CardNumber:  "378282246310005",
				Expiry:      "23/4",
				Amount:      20.5,
				Currency:    "GBP",
				Cvv:         342,
				PaymentType: protos.PaymentType_CARD,
				CardType:    protos.CardType_VISA,
			},
			status: protos.Status_APPROVED,
			reason: "approved and completed successfully",
			expected: protos.GetPaymentResponse{
				Ref:          refID,
				CardNumber:   "3782XXXXXXX0005",
				Amount:       20.5,
				Currency:     "GBP",
				PaymentType:  protos.PaymentType_CARD,
				Status:       protos.Status_APPROVED,
				StatusReason: "approved and completed successfully",
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			err := pgClient.AddPaymentInfo(ctx, tt.refID, tt.request, tt.status, tt.reason)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			//check that the data was inserted
			paymentInfo, err := pgClient.GetPaymentInfo(ctx, refID)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			assert.Equal(t, paymentInfo.CardNumber, "3782XXXXXXX0005")
			assert.Equal(t, paymentInfo.GetRef(), refID)
			assert.Equal(t, paymentInfo.GetAmount(), tt.request.GetAmount())
			assert.Equal(t, paymentInfo.GetCurrency(), tt.request.GetCurrency())
			assert.Equal(t, paymentInfo.GetStatus(), protos.Status_APPROVED)
			assert.Equal(t, paymentInfo.GetStatusReason(), "approved and completed successfully")

		})
	}
}

// Negative path as happy path tested above
func TestPgxStorage_GetPaymentInfo(t *testing.T) {
	ctx := context.Background()
	pool, err := postgres.CreatePgPool(ctx, "postgres://user1:123@localhost:5433/payments_test", 5, 1)
	if err != nil {
		log.WithError(err).Fatal("creating database pgClient")
	}

	refID := "825ca1787c9d4672991848a5bfbc10572"

	pgClient := postgres.New(pool)

	tests := []struct {
		name     string
		refID    string
		status   protos.Status
		reason   string
		expected protos.GetPaymentResponse
		err      error
	}{
		{
			name:   "payment does not exist",
			refID:  refID,
			status: protos.Status_APPROVED,
			reason: "approved and completed successfully",
			expected: protos.GetPaymentResponse{
				Ref:          "",
				CardNumber:   "",
				PaymentType:  protos.PaymentType_UNDEFINED,
				Status:       protos.Status_UNKNOWN,
				StatusReason: "transaction does not exist",
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			//check that the data was inserted
			paymentInfo, err := pgClient.GetPaymentInfo(ctx, refID)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			assert.Equal(t, paymentInfo.GetRef(), "")
			assert.Equal(t, paymentInfo.GetStatus(), protos.Status_UNKNOWN)
			assert.Equal(t, paymentInfo.GetStatusReason(), "transaction does not exist")

		})
	}
}
