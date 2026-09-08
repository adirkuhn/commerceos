package fx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/fx"
)

func TestExchangeRate_Convert(t *testing.T) {
	t.Run("applies the rate and rounds half away from zero", func(t *testing.T) {
		rate := fx.ExchangeRate{RateBasisPoints: 10876} // 1.0876
		require.Equal(t, int64(1088), rate.Convert(1000))
		require.Equal(t, int64(10876), rate.Convert(10000))
	})

	t.Run("zero stays zero", func(t *testing.T) {
		rate := fx.ExchangeRate{RateBasisPoints: 10876}
		require.Equal(t, int64(0), rate.Convert(0))
	})

	t.Run("negative amounts convert symmetrically (refunds)", func(t *testing.T) {
		rate := fx.ExchangeRate{RateBasisPoints: 10876}
		require.Equal(t, -rate.Convert(1000), rate.Convert(-1000))
	})

	t.Run("parity rate (10000) is a no-op", func(t *testing.T) {
		rate := fx.ExchangeRate{RateBasisPoints: 10000}
		require.Equal(t, int64(4599), rate.Convert(4599))
	})
}

func TestValidCurrencyCode(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"EUR", true},
		{"USD", true},
		{"eur", false},
		{"EU", false},
		{"EURO", false},
		{"", false},
		{"12R", false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, fx.ValidCurrencyCode(c.code), c.code)
	}
}
