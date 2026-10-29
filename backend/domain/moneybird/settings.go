// Package moneybird is the bookkeeping side of CommerceOS: quotes go to
// Moneybird as estimates, sales orders as external sales invoices (an
// invoice CommerceOS issued itself, booked in Moneybird; Moneybird neither
// numbers nor sends it) with the payment registered on it.
package moneybird

import (
	"fmt"
	"time"
)

// TaxRateMapping books a CommerceOS VAT rate on a Moneybird sales tax rate.
type TaxRateMapping struct {
	RateBasisPoints    int32
	MoneybirdTaxRateID string
}

// PaymentMethodMapping registers an order's payment method (e.g. Woo's
// "bacs") as a Moneybird payment method (e.g. "bank_transfer").
type PaymentMethodMapping struct {
	Method                 string
	MoneybirdPaymentMethod string
}

// Settings is one Moneybird connection's configuration.
type Settings struct {
	ConnectionID           string
	TenantID               string
	AdministrationID       string
	AdministrationName     string
	DefaultLedgerAccountID string
	// GuestContactID is the contact for an order with neither a customer
	// nor an email/company to find or create one by. Created on first use.
	GuestContactID string
	// AutoSync sends quotes and orders as they're created/changed. Off by
	// default: the shop may already book orders through another service,
	// so it sends the ones it needs by hand.
	AutoSync       bool
	TaxRates       []TaxRateMapping
	PaymentMethods []PaymentMethodMapping
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// rateToleranceBasisPoints: a rate derived from rounded cents (see
// RateBasisPoints) is a little off the real one -- 58 VAT on 640 is 9.06%.
const rateToleranceBasisPoints = 50

// Ready reports whether documents can be booked at all.
func (s Settings) Ready() error {
	if s.AdministrationID == "" || s.DefaultLedgerAccountID == "" {
		return ErrNotConfigured
	}
	return nil
}

// TaxRateID is the Moneybird tax rate for a VAT rate: the nearest mapped
// rate within rateToleranceBasisPoints. Never falls back to another rate --
// booking the wrong VAT is worse than not booking.
func (s Settings) TaxRateID(rateBasisPoints int32) (string, error) {
	best, bestDiff := "", int32(rateToleranceBasisPoints+1)
	for _, m := range s.TaxRates {
		diff := m.RateBasisPoints - rateBasisPoints
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			best, bestDiff = m.MoneybirdTaxRateID, diff
		}
	}
	if best == "" {
		return "", fmt.Errorf("%w: %s%%", ErrTaxRateNotMapped, formatRate(rateBasisPoints))
	}
	return best, nil
}

// PaymentMethod is the Moneybird payment method an order method maps to.
func (s Settings) PaymentMethod(method string) (string, bool) {
	for _, m := range s.PaymentMethods {
		if m.Method == method && m.MoneybirdPaymentMethod != "" {
			return m.MoneybirdPaymentMethod, true
		}
	}
	return "", false
}

// RateBasisPoints is the VAT rate an amount was taxed at, rounded to the
// nearest basis point (58 on 640 -> 906 = 9.06%).
func RateBasisPoints(taxMinorUnits, exclTaxMinorUnits int64) int32 {
	if exclTaxMinorUnits == 0 {
		return 0
	}
	return int32((taxMinorUnits*10000*2 + exclTaxMinorUnits) / (exclTaxMinorUnits * 2))
}

// FormatAmount is minor units as Moneybird's decimal string ("12.11").
func FormatAmount(minorUnits int64) string {
	sign := ""
	if minorUnits < 0 {
		sign, minorUnits = "-", -minorUnits
	}
	return fmt.Sprintf("%s%d.%02d", sign, minorUnits/100, minorUnits%100)
}

func formatRate(basisPoints int32) string {
	return fmt.Sprintf("%d.%02d", basisPoints/100, basisPoints%100)
}
