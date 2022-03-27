package server

import (
	"context"
	log "github.com/sirupsen/logrus"
	"payments_gateway/model"
	identifier "payments_gateway/utils"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	bank "payments_gateway/aquiring-bank"
	protos "payments_gateway/protos"
	"payments_gateway/storage"
)

type server struct {
	protos.UnimplementedPaymentsServer // server implementations must now be forward compatible
	dbClient                           storage.Client
	aqBank                             bank.Client
}

var _ protos.PaymentsServer = (*server)(nil)

var (
	_errInvalidParam        = status.Error(codes.InvalidArgument, "missing parameter")
	_errAddingPayment       = status.Error(codes.Internal, "error adding payment info")
	_errAGettingPaymentInfo = status.Error(codes.Internal, "error getting payment info")
)

// New - grpc server constructor
func New(dbClient storage.Client, aqBankClient bank.Client) *server {
	return &server{
		dbClient: dbClient,
		aqBank:   aqBankClient,
	}
}

// ProcessPayment processes payments made to the payments gateway
func (s *server) ProcessPayment(ctx context.Context, request *protos.ProcessPaymentRequest) (*protos.ProcessPaymentResponse, error) {
	// validation of input parameters
	if !validParams(request.GetAmount(), request.GetCurrency(), request.GetCardNumber(), request.GetCvv(), request.GetExpiry()) {
		log.WithField("request", request).Warn("request contains invalid parameters")

		return nil, _errInvalidParam
	}

	refID := identifier.NewUUID()

	// validate the card info
	if ok, err := s.aqBank.Validate(ctx, model.ConvertToCardDetails(request)); !ok {
		log.WithField("request", request).WithError(err).Error("invalid card details")

		return &protos.ProcessPaymentResponse{
			Error: &protos.Error{
				Reason: err.Error(),
			},
		}, status.Error(codes.InvalidArgument, "validating payment")
	}

	// Authorise the users cars details and funds for the purchase
	code, reason, err := s.aqBank.Authorize(ctx, model.ConvertToTransaction(refID, request))
	if err != nil {
		log.WithField("request", request).WithError(err).Error("authorize transaction")

		return nil, status.Error(codes.Internal, "authorize transaction")
	}

	status := determineStatus(code)

	// add the payment info to DB
	// TODO refactor and use model types instead of request
	if err := s.dbClient.AddPaymentInfo(ctx, refID, request, status, reason); err != nil {
		return nil, _errAddingPayment
	}

	return &protos.ProcessPaymentResponse{
		Reference:    refID,
		Status:       status,
		StatusReason: reason,
	}, nil
}

// GetPayment retrieves payments previously made to the payments gateway
func (s *server) GetPayment(ctx context.Context, request *protos.GetPaymentRequest) (*protos.GetPaymentResponse, error) {
	// validation of input parameters
	if !validParams(request.GetRef()) {
		log.WithField("request", request).Warn("request contains invalid parameters")

		return nil, _errInvalidParam
	}

	resp, err := s.dbClient.GetPaymentInfo(ctx, request.GetRef())
	if err != nil {
		return nil, _errAGettingPaymentInfo
	}

	return resp, err
}

func (s *server) Register(grpcService *grpc.Server) {
	protos.RegisterPaymentsServer(grpcService, s)
}

// valid reports whether v is the zero value for its type.
func validParams(params ...interface{}) bool {
	for _, p := range params {
		if reflect.ValueOf(p).IsZero() {
			return false
		}
	}

	return true
}

//determineStatus Interpret codes returned from the payment gateway
func determineStatus(code string) protos.Status {
	switch code {
	case "00":
		return protos.Status_APPROVED
	case "06", "39", "12":
		return protos.Status_REJECTED
	case "19":
		return protos.Status_PENDING
	case "20":
		return protos.Status_COMPLETED
	default:
		return protos.Status_PENDING
	}
}
