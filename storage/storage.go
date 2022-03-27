package storage

import (
	"context"
	protos "payments_gateway/protos"
)

// Client is the interface for storage operations
type Client interface {
	AddPaymentInfo(ctx context.Context, refID string, request *protos.ProcessPaymentRequest, code protos.Status, reason string) error
	GetPaymentInfo(ctx context.Context, refId string) (*protos.GetPaymentResponse, error)
}
