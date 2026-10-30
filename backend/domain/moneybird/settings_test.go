package moneybird_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/moneybird"
)

func mapped() moneybird.Settings {
	return moneybird.Settings{
		AdministrationID: "adm-1", DefaultLedgerAccountID: "led-1",
		TaxRates: []moneybird.TaxRateMapping{
			{RateBasisPoints: 2100, MoneybirdTaxRateID: "mb-21"},
			{RateBasisPoints: 900, MoneybirdTaxRateID: "mb-9"},
			{RateBasisPoints: 0, MoneybirdTaxRateID: "mb-0"},
		},
		PaymentMethods: []moneybird.PaymentMethodMapping{{Method: "bacs", MoneybirdPaymentMethod: "bank_transfer"}},
	}
}

func TestTaxRateID_exactRate(t *testing.T) {
	id, err := mapped().TaxRateID(900)
	require.NoError(t, err)
	assert.Equal(t, "mb-9", id)
}

// A rate derived from rounded cents (58 VAT on 640 = 9.06%) still finds 9%.
func TestTaxRateID_roundedRateFindsNearestMappedRate(t *testing.T) {
	id, err := mapped().TaxRateID(906)
	require.NoError(t, err)
	assert.Equal(t, "mb-9", id)
}

// Never a silent fallback to some other rate: booking the wrong VAT is worse than not booking.
func TestTaxRateID_unmappedRateFails(t *testing.T) {
	_, err := mapped().TaxRateID(1300)
	assert.True(t, errors.Is(err, moneybird.ErrTaxRateNotMapped))
}

func TestPaymentMethod(t *testing.T) {
	got, ok := mapped().PaymentMethod("bacs")
	assert.True(t, ok)
	assert.Equal(t, "bank_transfer", got)

	_, ok = mapped().PaymentMethod("stripe")
	assert.False(t, ok)
}

func TestReady(t *testing.T) {
	require.NoError(t, mapped().Ready())

	s := mapped()
	s.AdministrationID = ""
	assert.True(t, errors.Is(s.Ready(), moneybird.ErrNotConfigured))

	s = mapped()
	s.DefaultLedgerAccountID = ""
	assert.True(t, errors.Is(s.Ready(), moneybird.ErrNotConfigured))
}

func TestRateBasisPoints(t *testing.T) {
	assert.Equal(t, int32(906), moneybird.RateBasisPoints(58, 640))
	assert.Equal(t, int32(2100), moneybird.RateBasisPoints(210, 1000))
	assert.Equal(t, int32(0), moneybird.RateBasisPoints(0, 1000))
	assert.Equal(t, int32(0), moneybird.RateBasisPoints(0, 0))
}

func TestFormatAmount(t *testing.T) {
	assert.Equal(t, "12.11", moneybird.FormatAmount(1211))
	assert.Equal(t, "0.05", moneybird.FormatAmount(5))
	assert.Equal(t, "-3.00", moneybird.FormatAmount(-300))
}

func TestPaymentFor(t *testing.T) {
	s := mapped()

	method, register, err := s.PaymentFor("bacs")
	require.NoError(t, err)
	assert.True(t, register)
	assert.Equal(t, "bank_transfer", method)

	_, register, err = s.PaymentFor("")
	require.NoError(t, err)
	assert.False(t, register, "no method on the order: book it, leave the payment to Moneybird")

	_, _, err = s.PaymentFor("stripe")
	assert.True(t, errors.Is(err, moneybird.ErrPaymentMethodNotMapped))

	s.PaymentMethods = append(s.PaymentMethods, moneybird.PaymentMethodMapping{Method: "", MoneybirdPaymentMethod: "cash"})
	method, register, err = s.PaymentFor("")
	require.NoError(t, err)
	assert.True(t, register)
	assert.Equal(t, "cash", method)
}
