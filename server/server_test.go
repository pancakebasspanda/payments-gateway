package server

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"payments_gateway/aquiring-bank/mocks"
	"payments_gateway/model"
	protos "payments_gateway/protos"
	"payments_gateway/storage/mocks"
)

func Test_server_ProcessPayment(t *testing.T) {
	mockController := gomock.NewController(t)

	storageMock := mock_storage.NewMockClient(mockController)

	bankMock := mock_bank.NewMockClient(mockController)

	defer mockController.Finish()

	validationErr := fmt.Errorf("failed validation: invalid card number")

	req := &protos.ProcessPaymentRequest{
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
	}

	type args struct {
		request             *protos.ProcessPaymentRequest
		storageMockOutcomes func(storageMock *mock_storage.MockClient)
		BankMockOutcomes    func(bankMock *mock_bank.MockClient)
	}

	var tests = []struct {
		name string
		args args
		want *protos.ProcessPaymentResponse
		err  error
	}{
		{
			name: "successfully process payment",
			args: args{
				request: req,
				storageMockOutcomes: func(storageMock *mock_storage.MockClient) {
					storageMock.EXPECT().
						AddPaymentInfo(gomock.Any(), gomock.Any(), req, protos.Status_APPROVED, "approved and completed successfully").
						Times(1).
						Return(nil)
				},
				BankMockOutcomes: func(bankMock *mock_bank.MockClient) {
					bankMock.EXPECT().
						Validate(gomock.Any(), model.ConvertToCardDetails(req)).
						Times(1).
						Return(true, nil)
					bankMock.EXPECT().
						Authorize(gomock.Any(), gomock.Any()). // TODO cater for changing refID, generated in server to avoid gomock.Any()
						Times(1).
						Return("00", "approved and completed successfully", nil)
				},
			},
			want: &protos.ProcessPaymentResponse{
				Reference:    "825ca1787c9d4672991848a5bfbc1057",
				Status:       protos.Status_APPROVED,
				StatusReason: "approved and completed successfully",
			},
			err: nil,
		},
		{
			name: "fails validation",
			args: args{
				request: req,
				storageMockOutcomes: func(storageMock *mock_storage.MockClient) {
				},
				BankMockOutcomes: func(bankMock *mock_bank.MockClient) {
					bankMock.EXPECT().
						Validate(gomock.Any(), model.ConvertToCardDetails(req)).
						Times(1).
						Return(false, validationErr)
				},
			},
			err: fmt.Errorf("rpc error: code = InvalidArgument desc = validating payment"),
		},
		{
			name: "Rejected auth",
			args: args{
				request: req,
				storageMockOutcomes: func(storageMock *mock_storage.MockClient) {
					storageMock.EXPECT().
						AddPaymentInfo(gomock.Any(), gomock.Any(), req, protos.Status_REJECTED, "transaction error").
						Times(1).
						Return(nil)
				},
				BankMockOutcomes: func(bankMock *mock_bank.MockClient) {
					bankMock.EXPECT().
						Validate(gomock.Any(), model.ConvertToCardDetails(req)).
						Times(1).
						Return(true, nil)
					bankMock.EXPECT().
						Authorize(gomock.Any(), gomock.Any()). // TODO cater for changing refID, generated in server to avoid gomock.Any()
						Times(1).
						Return("06", "transaction error", nil)
				},
			},
			want: &protos.ProcessPaymentResponse{
				Reference:    "825ca1787c9d4672991848a5bfbc1057",
				Status:       protos.Status_REJECTED,
				StatusReason: "transaction error",
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.storageMockOutcomes(storageMock)
			tt.args.BankMockOutcomes(bankMock)

			s := New(storageMock, bankMock)

			got, err := s.ProcessPayment(context.Background(), tt.args.request)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			assert.NotEmpty(t, got.Reference, tt.want.Reference)

			assert.Equal(t, got.StatusReason, tt.want.StatusReason)

			assert.Equal(t, got.Status, tt.want.Status)
		})
	}
}

func Test_server_GetPayment(t *testing.T) {
	mockController := gomock.NewController(t)

	storageMock := mock_storage.NewMockClient(mockController)

	bankMock := mock_bank.NewMockClient(mockController)

	defer mockController.Finish()

	refID := "825ca1787c9d4672991848a5bfbc1057"

	const layout = "Jan 2, 2006 at 3:04pm (MST)"

	tm, err := time.Parse(layout, "Feb 4, 2014 at 6:05pm (PST)")

	if err != nil {
		t.Fail()
	}

	resp := &protos.GetPaymentResponse{
		Ref:              refID,
		CardNumber:       "3782XXXXXXX0005",
		Amount:           20.5,
		Currency:         "GBP",
		PaymentType:      protos.PaymentType_CARD,
		Status:           protos.Status_APPROVED,
		StatusReason:     "approved and completed successfully",
		UpdatedTimestamp: timestamppb.New(tm),
		BillingDetails: &protos.BillingDetails{
			Name:          "Bruce",
			Surname:       "Wayne",
			Email:         "iam@batman.com",
			Phone:         "0789825678",
			AddressLine_1: "wayne manor",
			AddressLine_2: "Gotham city",
			Postcode:      "G15 2D",
		},
		InsertTimestamp: timestamppb.New(tm),
	}

	type args struct {
		request             *protos.GetPaymentRequest
		storageMockOutcomes func(storageMock *mock_storage.MockClient)
		BankMockOutcomes    func(bankMock *mock_bank.MockClient)
	}
	tests := []struct {
		name string
		args args
		want *protos.GetPaymentResponse
		err  error
	}{
		{
			name: "successfully get payment details",
			args: args{
				request: &protos.GetPaymentRequest{
					Ref: refID,
				},
				storageMockOutcomes: func(storageMock *mock_storage.MockClient) {
					storageMock.EXPECT().
						GetPaymentInfo(gomock.Any(), refID).
						Times(1).
						Return(resp, nil)
				},
				BankMockOutcomes: func(bankMock *mock_bank.MockClient) {
				},
			},
			want: resp,
			err:  nil,
		},
		{
			name: "reference transaction not found",
			args: args{
				request: &protos.GetPaymentRequest{
					Ref: refID,
				},
				storageMockOutcomes: func(storageMock *mock_storage.MockClient) {
					storageMock.EXPECT().
						GetPaymentInfo(gomock.Any(), refID).
						Times(1).
						Return(&protos.GetPaymentResponse{Ref: refID, Status: protos.Status_UNKNOWN, PaymentType: protos.PaymentType_UNDEFINED, StatusReason: "transaction does not exist"}, nil)
				},
				BankMockOutcomes: func(bankMock *mock_bank.MockClient) {
				},
			},
			want: &protos.GetPaymentResponse{
				Ref:          "825ca1787c9d4672991848a5bfbc1057",
				PaymentType:  protos.PaymentType_UNDEFINED,
				Status:       protos.Status_UNKNOWN,
				StatusReason: "transaction does not exist",
			},
			err: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.args.storageMockOutcomes(storageMock)
			tt.args.BankMockOutcomes(bankMock)

			s := New(storageMock, bankMock)

			got, err := s.GetPayment(context.Background(), tt.args.request)
			if err != nil {
				assert.Equal(t, err.Error(), tt.err.Error())

				return
			}

			assert.NotEmpty(t, got, tt.want)

		})
	}
}
