//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSalesOrderRepo_Integration_ShipDueOnOrBefore proves the "orders to
// ship today" filter: ship_by_date on/before the date, still unshipped
// (processing/on-hold, picked or packed included), never POS, never
// another tenant's -- with Count matching List, read as the RLS-bound app
// role.
func TestSalesOrderRepo_Integration_ShipDueOnOrBefore(t *testing.T) {
	db, superDSN := newIntegrationDBAndDSN(t)
	appDB := newAppRoleDB(t, superDSN)
	appRepo := repositories.NewSalesOrderRepo(appDB)
	_, ctx := newTestTenant()
	_, otherCtx := newTestTenant()
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	_, otherWarehouseID := seedProductSKUWarehouse(t, db, otherCtx)
	repo := repositories.NewSalesOrderRepo(db)

	day := func(s string) *time.Time {
		d, err := time.Parse(time.DateOnly, s)
		require.NoError(t, err)
		return &d
	}
	type spec struct {
		extID  string
		source omsDomain.SalesOrderSource
		status omsDomain.SalesOrderStatus
		packed bool
		shipBy *time.Time
	}
	seed := func(ctx context.Context, warehouseID string, specs []spec) map[string]string {
		t.Helper()
		ids := map[string]string{}
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			now := time.Now()
			for _, s := range specs {
				o := omsDomain.SalesOrder{
					TenantID: tenant.IDFromContext(ctx), Source: s.source, ExternalOrderID: s.extID, WarehouseID: warehouseID,
					CurrencyCode: "EUR", Status: s.status, Packed: s.packed, Fulfilled: s.packed,
					ShipByDate: s.shipBy, OccurredAt: now, CreatedAt: now,
				}
				if err := repo.Create(ctx, &o); err != nil {
					return err
				}
				ids[s.extID] = o.ID
			}
			return nil
		})
		require.NoError(t, err)
		return ids
	}
	bol, pos := omsDomain.SalesOrderSourceBol, omsDomain.SalesOrderSourcePOS
	processing := omsDomain.SalesOrderStatusProcessing
	ids := seed(ctx, warehouseID, []spec{
		{extID: "sd-today", source: bol, status: processing, shipBy: day("2026-10-12")},                              // due
		{extID: "sd-overdue", source: bol, status: omsDomain.SalesOrderStatusOnHold, shipBy: day("2026-10-10")},      // due
		{extID: "sd-packed", source: bol, status: processing, packed: true, shipBy: day("2026-10-12")},               // due: not handed over yet
		{extID: "sd-tomorrow", source: bol, status: processing, shipBy: day("2026-10-13")},                           // not yet
		{extID: "sd-shipped", source: bol, status: omsDomain.SalesOrderStatusShipped, shipBy: day("2026-10-11")},     // left
		{extID: "sd-cancelled", source: bol, status: omsDomain.SalesOrderStatusCancelled, shipBy: day("2026-10-11")}, // never
		{extID: "sd-nodate", source: omsDomain.SalesOrderSourceWooCommerce, status: processing},                      // no deadline
		{extID: "sd-pos", source: pos, status: processing, shipBy: day("2026-10-01")},                                // POS never ships
	})
	seed(otherCtx, otherWarehouseID, []spec{
		{extID: "sd-other", source: bol, status: processing, shipBy: day("2026-10-01")},
	})

	filter := omsDomain.ListSalesOrdersFilter{ShipDueOnOrBefore: day("2026-10-12")}
	var count, listTotal int64
	var listed []omsDomain.SalesOrder
	var stored *omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, appDB, func(ctx context.Context) error {
		var err error
		if count, err = appRepo.Count(ctx, filter); err != nil {
			return err
		}
		if listed, listTotal, err = appRepo.List(ctx, filter); err != nil {
			return err
		}
		stored, err = appRepo.FindByID(ctx, ids["sd-today"])
		return err
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
	assert.Equal(t, listTotal, count, "Count must match List's total for the same filter")
	got := map[string]bool{}
	for _, o := range listed {
		got[o.ExternalOrderID] = true
	}
	assert.Equal(t, map[string]bool{"sd-today": true, "sd-overdue": true, "sd-packed": true}, got)
	require.NotNil(t, stored.ShipByDate, "ship_by_date must round-trip")
	assert.Equal(t, "2026-10-12", stored.ShipByDate.Format(time.DateOnly))
}
