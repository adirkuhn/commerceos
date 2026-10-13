package inventory_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestResolveUnitCost_prefersTheLotsOwnCost(t *testing.T) {
	lot, average := int64(210), int64(250)

	cost, source := invDomain.ResolveUnitCost(&lot, &average)

	require.NotNil(t, cost)
	assert.Equal(t, int64(210), *cost)
	assert.Equal(t, invDomain.UnitCostSourceBatch, source)
}

func TestResolveUnitCost_fallsBackToTheAverage(t *testing.T) {
	average := int64(250)

	cost, source := invDomain.ResolveUnitCost(nil, &average)

	require.NotNil(t, cost)
	assert.Equal(t, int64(250), *cost)
	assert.Equal(t, invDomain.UnitCostSourceSKU, source)
}

func TestResolveUnitCost_unknownWithoutEither(t *testing.T) {
	cost, source := invDomain.ResolveUnitCost(nil, nil)

	assert.Nil(t, cost)
	assert.Equal(t, invDomain.UnitCostSourceNone, source)
}
