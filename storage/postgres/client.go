package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	protos "payments_gateway/protos"
	"payments_gateway/storage"
)

type PgPool interface {
	pgxtype.Querier
	Close()
}

type PgxStorage struct {
	pool PgPool
}

var _ storage.Client = (*PgxStorage)(nil)

var (
	errInsertingPaymentInfo = errors.New("error inserting payment info")
)

func New(pool PgPool) *PgxStorage {
	return &PgxStorage{pool: pool}
}

// AddPaymentInfo adds payment information from the transactions to the DB
func (p *PgxStorage) AddPaymentInfo(ctx context.Context, refID string, request *protos.ProcessPaymentRequest, status protos.Status, reason string) error {
	maskedCard := maskCardNumber(request.GetCardNumber(), 'X')

	res, err := p.pool.Exec(ctx,
		_insertPaymentInfo,
		convertStringToPgType(refID),
		convertStringToPgType(request.GetBillingDetails().GetName()),
		convertStringToPgType(request.GetBillingDetails().GetSurname()),
		convertStringToPgType(request.GetBillingDetails().GetEmail()),
		convertStringToPgType(request.GetBillingDetails().GetPhone()),
		convertStringToPgType(request.GetBillingDetails().GetAddressLine_1()),
		convertStringToPgType(request.GetBillingDetails().GetAddressLine_2()),
		convertStringToPgType(request.GetBillingDetails().GetPostcode()),
		convertStringToPgType(maskedCard),
		convertStringToPgType(request.GetCurrency()),
		convertFloatToPgType(request.GetAmount()),
		convertEnumToPgType(request.GetPaymentType()),
		convertEnumToPgType(status),
		convertStringToPgType(reason),
	)

	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return errInsertingPaymentInfo
	}

	return nil
}

// GetPaymentInfo retrieves payment information from the transactions to the DB
func (p *PgxStorage) GetPaymentInfo(ctx context.Context, referenceId string) (*protos.GetPaymentResponse, error) {
	id := pgtype.Text{}

	err := id.Set(referenceId)
	if err != nil {
		return nil, err
	}

	result := p.pool.QueryRow(ctx, _getPaymentInfo, id)

	var refId, name, surname, email, phone, address1, address2, postcode, cardNo, currency, status_reason pgtype.Varchar

	var amount pgtype.Float8

	var status, paymentType pgtype.Varchar

	var insertTime, updatedTime pgtype.Timestamp

	if err := result.Scan(&refId, &name, &surname, &email, &phone, &address1, &address2, &postcode, &cardNo, &currency, &amount, &paymentType, &status, &status_reason, &insertTime, &updatedTime); err != nil {
		if err.Error() == "no rows in result set" {
			return &protos.GetPaymentResponse{Ref: refId.String, Status: protos.Status_UNKNOWN, PaymentType: protos.PaymentType_UNDEFINED, StatusReason: "transaction does not exist"}, nil
		}
		return nil, err
	}

	return &protos.GetPaymentResponse{
		Ref:              refId.String,
		CardNumber:       cardNo.String,
		Amount:           amount.Float,
		Currency:         currency.String,
		PaymentType:      protos.PaymentType(protos.PaymentType_value[paymentType.String]),
		Status:           protos.Status(protos.PaymentType_value[status.String]),
		StatusReason:     status_reason.String,
		UpdatedTimestamp: timestamppb.New(updatedTime.Time),
		InsertTimestamp:  timestamppb.New(insertTime.Time),
		BillingDetails: &protos.BillingDetails{
			Name:          name.String,
			Surname:       surname.String,
			Email:         email.String,
			Phone:         phone.String,
			AddressLine_1: address1.String,
			AddressLine_2: address2.String,
			Postcode:      postcode.String,
		},
	}, nil

}

// CreatePgPool a pgx connection pool to connect and perform operations on the DB
func CreatePgPool(ctx context.Context, postgresURL string, poolMaxConnections int, poolMinConnections int) (PgPool, error) {
	log.WithFields(
		log.Fields{
			"postgres url":    postgresURL,
			"max connections": poolMaxConnections,
			"min connections": poolMinConnections,
		}).Debug("connecting to postgres")

	config, err := pgxpool.ParseConfig(postgresURL)

	if err != nil {
		return nil, fmt.Errorf("error parsing config %w", err)
	}

	config.MaxConns = int32(poolMaxConnections)
	config.MinConns = int32(poolMinConnections)
	config.ConnConfig.RuntimeParams = map[string]string{"standard_conforming_strings": "on"}
	config.ConnConfig.LogLevel = pgxLevel()
	config.ConnConfig.Logger = &pgLogger{}
	config.ConnConfig.PreferSimpleProtocol = true

	return pgxpool.ConnectConfig(ctx, config)
}

// Close - closes all connections in the pool
func (p *PgxStorage) Close() {
	p.pool.Close()
}

func pgxLevel() pgx.LogLevel {
	if log.GetLevel() == log.DebugLevel {
		return pgx.LogLevelTrace
	}
	return pgx.LogLevelWarn
}

func convertStringToPgType(value string) pgtype.Varchar {
	pgValue := pgtype.Varchar{String: value, Status: pgtype.Present}
	if value == "" {
		pgValue.Status = pgtype.Null
	}

	return pgValue
}

func convertFloatToPgType(value float64) pgtype.Float8 {
	pgValue := pgtype.Float8{Float: value, Status: pgtype.Present}
	if value == 0 {
		pgValue.Status = pgtype.Null
	}

	return pgValue
}

func convertEnumToPgType(value fmt.Stringer) pgtype.Varchar {
	return pgtype.Varchar{
		String: value.String(),
		Status: pgtype.Present,
	}
}

// maskCardNumber masks the all card numbers apart from the first
// and the last four numbers
func maskCardNumber(in string, r rune) string {
	out := []rune(in)
	for i := 4; i < (len(out) - 4); i++ {
		out[i] = r
	}

	return string(out)
}
