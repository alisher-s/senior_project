package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v76"
	stripeSession "github.com/stripe/stripe-go/v76/checkout/session"
)

type StripeClient struct {
	secretKey  string
	successURL string
	cancelURL  string
}

func NewStripeClient(secretKey, successURL, cancelURL string) *StripeClient {
	return &StripeClient{
		secretKey:  secretKey,
		successURL: successURL,
		cancelURL:  cancelURL,
	}
}

// CreateCheckoutSession returns (sessionID, sessionURL, error).
// amount is in the smallest currency unit (cents for USD, tenge for KZT which is zero-decimal).
func (c *StripeClient) CreateCheckoutSession(ctx context.Context, amount int64, currency, productName string) (string, string, error) {
	stripe.Key = c.secretKey

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(strings.ToLower(currency)),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(productName),
					},
					UnitAmount: stripe.Int64(amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(c.successURL),
		CancelURL:  stripe.String(c.cancelURL),
	}

	sess, err := stripeSession.New(params)
	if err != nil {
		return "", "", fmt.Errorf("stripe create session: %w", err)
	}
	return sess.ID, sess.URL, nil
}
