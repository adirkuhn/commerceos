package pms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestParseStockLevel(t *testing.T) {
	for in, want := range map[string]pmsDomain.StockLevel{
		"":    pmsDomain.StockLevelAny,
		"low": pmsDomain.StockLevelLow,
		"out": pmsDomain.StockLevelOut,
	} {
		got, err := pmsDomain.ParseStockLevel(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}

	_, err := pmsDomain.ParseStockLevel("high")
	assert.ErrorIs(t, err, pmsDomain.ErrInvalidStockLevel)
}

func TestValidateLowStockThreshold(t *testing.T) {
	zero, ten, negative := 0, 10, -1
	assert.NoError(t, pmsDomain.ValidateLowStockThreshold(nil), "nil clears the override")
	assert.NoError(t, pmsDomain.ValidateLowStockThreshold(&zero), "0 = never low for this product")
	assert.NoError(t, pmsDomain.ValidateLowStockThreshold(&ten))
	assert.ErrorIs(t, pmsDomain.ValidateLowStockThreshold(&negative), pmsDomain.ErrInvalidLowStockThreshold)
}
