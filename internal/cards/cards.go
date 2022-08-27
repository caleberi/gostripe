package cards

import (
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/sub"
)

type Status int

type Card struct {
	Secret   string
	Key      string
	Currency string
}

type Transaction struct {
	TransactionStatusID Status
	Amount              int
	LastFour            string
	Currency            string
	BankReturnCode      string
}

func (c *Card) Charge(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	return c.CreatePaymentIntent(currency, amount)
}

func (card *Card) CreatePaymentIntent(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	stripe.Key = card.Secret
	// create payment intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
	}

	paymentIntent, err := paymentintent.New(params)
	if err != nil {
		msg := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = cardErrorMessage(stripeErr.Code)
		}
		return nil, msg, nil
	}
	return paymentIntent, "", nil
}

func cardErrorMessage(code stripe.ErrorCode) string {
	var msg string
	switch code {
	case stripe.ErrorCodeCardDeclined:
		msg = "Your card was declined"
	case stripe.ErrorCodeExpiredCard:
		msg = "Your card is expired "
	case stripe.ErrorCodeIncorrectCVC:
		msg = "Incorrect CVC code"
	case stripe.ErrorCodeAmountTooLarge:
		msg = "The amount is too large to charge to your card"
	case stripe.ErrorCodeAmountTooSmall:
		msg = "The amount is too small to charge to your card"
	case stripe.ErrorCodeIncorrectZip:
		msg = "Incorrect zip/postal code"
	case stripe.ErrorCodeBalanceInsufficient:
		msg = "Insufficent balance"
	case stripe.ErrorCodePostalCodeInvalid:
		msg = "Your postal code is invalid"
	default:
		msg = "Your card was declined"
	}
	return msg
}

// GetPaymentMethod retrieves the payment intent by payment id
func (card *Card) GetPaymentMethod(s string) (*stripe.PaymentMethod, error) {
	stripe.Key = card.Secret

	pm, err := paymentmethod.Get(s, nil)

	if err != nil {
		return nil, err
	}
	return pm, nil
}

// RetrivePaymentIntent gets existing payment intent
func (card *Card) RetrivePaymentIntent(id string) (*stripe.PaymentIntent, error) {
	stripe.Key = card.Secret

	pi, err := paymentintent.Get(id, nil)

	if err != nil {
		return nil, err
	}
	return pi, nil
}

func (card *Card) SubscribeToPlan(customer *stripe.Customer, plan, email, last4, cardType string) (*stripe.Subscription, error) {
	stripeCustomerID := customer.ID
	items := []*stripe.SubscriptionItemsParams{
		{Plan: stripe.String(plan)},
	}
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(stripeCustomerID),
		Items:    items,
	}

	params.AddMetadata("last_four", last4)
	params.AddMetadata("card_type", cardType)
	params.AddExpand("latest_invoice.payment_intent")
	subscription, err := sub.New(params)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (card *Card) CreateCustomer(pm, email string) (*stripe.Customer, string, error) {
	stripe.Key = card.Secret
	customerParams := &stripe.CustomerParams{
		PaymentMethod: stripe.String(pm),
		Email:         stripe.String(email),
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm),
		},
	}

	customer, err := customer.New(customerParams)

	if err != nil {
		msg := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = cardErrorMessage(stripeErr.Code)
		}
		return nil, msg, err
	}

	return customer, "", nil
}
