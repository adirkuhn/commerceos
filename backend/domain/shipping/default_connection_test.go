package shipping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

var (
	day1 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	day2 = day1.AddDate(0, 0, 1)
	day3 = day1.AddDate(0, 0, 2)
)

func TestResolveDefaultConnection_noConnectionsHasNoDefault(t *testing.T) {
	assert.Equal(t, "", shippingDomain.ResolveDefaultConnection(nil, strPtr("gone")))
}

func TestResolveDefaultConnection_singleConnectionIsTheDefault(t *testing.T) {
	conns := []shippingDomain.ConnectionCandidate{{ID: "only", CreatedAt: day2}}

	assert.Equal(t, "only", shippingDomain.ResolveDefaultConnection(conns, nil))
}

func TestResolveDefaultConnection_configuredDefaultWins(t *testing.T) {
	conns := []shippingDomain.ConnectionCandidate{{ID: "old", CreatedAt: day1}, {ID: "picked", CreatedAt: day3}}

	assert.Equal(t, "picked", shippingDomain.ResolveDefaultConnection(conns, strPtr("picked")))
}

func TestResolveDefaultConnection_unsetFallsBackToOldest(t *testing.T) {
	conns := []shippingDomain.ConnectionCandidate{{ID: "newer", CreatedAt: day3}, {ID: "oldest", CreatedAt: day1}, {ID: "mid", CreatedAt: day2}}

	assert.Equal(t, "oldest", shippingDomain.ResolveDefaultConnection(conns, nil))
}

func TestResolveDefaultConnection_configuredButNoLongerShippingFallsBackToOldest(t *testing.T) {
	conns := []shippingDomain.ConnectionCandidate{{ID: "newer", CreatedAt: day3}, {ID: "oldest", CreatedAt: day1}}

	assert.Equal(t, "oldest", shippingDomain.ResolveDefaultConnection(conns, strPtr("deleted")))
}

func TestResolveDefaultConnection_sameCreatedAtBreaksTieByID(t *testing.T) {
	conns := []shippingDomain.ConnectionCandidate{{ID: "b", CreatedAt: day1}, {ID: "a", CreatedAt: day1}}

	assert.Equal(t, "a", shippingDomain.ResolveDefaultConnection(conns, nil))
}
