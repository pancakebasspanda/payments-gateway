package model

import protos "payments_gateway/protos"

type Card struct {
	Name     string
	Surname  string
	Postcode string
	CardType string
	CardNum  string
	Expiry   string
	Cvv      int32
}

type Transaction struct {
	RefID    string
	Card     Card
	Amount   float64
	Currency string
}

func ConvertToCardDetails(request *protos.ProcessPaymentRequest) Card {
	return Card{
		Name:     request.GetBillingDetails().GetName(),
		Surname:  request.GetBillingDetails().GetSurname(),
		Postcode: request.GetBillingDetails().GetPostcode(),
		CardType: request.GetCardType().String(),
		CardNum:  request.GetCardNumber(),
		Expiry:   request.GetExpiry(),
		Cvv:      request.GetCvv(),
	}
}

func ConvertToTransaction(refID string, request *protos.ProcessPaymentRequest) Transaction {
	return Transaction{
		RefID:    refID,
		Card:     ConvertToCardDetails(request),
		Amount:   request.GetAmount(),
		Currency: request.GetCurrency(),
	}
}
