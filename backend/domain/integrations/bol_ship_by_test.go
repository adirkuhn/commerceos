package integrations_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

func bolDate(s string) *time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return &d
}

func openItem(latest, exact *time.Time) integrations.BolOrderItem {
	return integrations.BolOrderItem{Quantity: 1, LatestDeliveryDate: latest, ExactDeliveryDate: exact}
}

func TestBolOrder_ShipByDate_isTheDayBeforeTheLatestDeliveryDate(t *testing.T) {
	// 2026-10-15 is a Thursday -> hand to the carrier on Wednesday.
	o := integrations.BolOrder{OrderItems: []integrations.BolOrderItem{openItem(bolDate("2026-10-15"), nil)}}

	got := o.ShipByDate()

	require.NotNil(t, got)
	assert.Equal(t, "2026-10-14", got.Format(time.DateOnly))
	assert.Equal(t, time.UTC, got.Location())
}

func TestBolOrder_ShipByDate_exactDeliveryDateIsUsedWhenSet(t *testing.T) {
	o := integrations.BolOrder{OrderItems: []integrations.BolOrderItem{openItem(nil, bolDate("2026-10-16"))}}

	got := o.ShipByDate()

	require.NotNil(t, got)
	assert.Equal(t, "2026-10-15", got.Format(time.DateOnly))
}

func TestBolOrder_ShipByDate_mondayDeliveryShipsOnSaturday(t *testing.T) {
	// 2026-10-19 is a Monday; the day before is a Sunday, when carriers
	// don't collect, so the parcel must leave on Saturday.
	o := integrations.BolOrder{OrderItems: []integrations.BolOrderItem{openItem(bolDate("2026-10-19"), nil)}}

	got := o.ShipByDate()

	require.NotNil(t, got)
	assert.Equal(t, "2026-10-17", got.Format(time.DateOnly))
}

func TestBolOrder_ShipByDate_earliestOpenItemWins(t *testing.T) {
	cancelled := openItem(bolDate("2026-10-13"), nil)
	cancelled.QuantityCancelled = 1
	o := integrations.BolOrder{OrderItems: []integrations.BolOrderItem{
		openItem(bolDate("2026-10-16"), nil),
		openItem(bolDate("2026-10-15"), nil),
		cancelled,
	}}

	got := o.ShipByDate()

	require.NotNil(t, got)
	assert.Equal(t, "2026-10-14", got.Format(time.DateOnly))
}

func TestBolOrder_ShipByDate_nilWithoutAnyDeliveryDate(t *testing.T) {
	shipped := openItem(bolDate("2026-10-15"), nil)
	shipped.QuantityShipped = 1
	o := integrations.BolOrder{OrderItems: []integrations.BolOrderItem{openItem(nil, nil), shipped}}

	assert.Nil(t, o.ShipByDate())
	assert.Nil(t, integrations.BolOrder{}.ShipByDate())
}
