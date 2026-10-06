package procurement_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/domain/procurement/planning"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

type stubSupplierFinder struct {
	err error
}

func (s *stubSupplierFinder) FindByID(_ context.Context, id string) (*suppliersDomain.Supplier, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &suppliersDomain.Supplier{ID: id, Name: "Bauducco NL"}, nil
}

type stubTenantSettings struct{ currency string }

func (s *stubTenantSettings) Get(context.Context) (*settingsDomain.TenantSettings, error) {
	return &settingsDomain.TenantSettings{DefaultCurrency: s.currency}, nil
}

type planOutbox struct {
	subjects []string
	payloads [][]byte
	err      error
}

func (o *planOutbox) Enqueue(_ context.Context, subject, _ string, payload []byte, _ time.Time) error {
	if o.err != nil {
		return o.err
	}
	o.subjects = append(o.subjects, subject)
	o.payloads = append(o.payloads, payload)
	return nil
}

type planOrderFixture struct {
	po     *stubPurchaseOrderRepo
	lines  *stubLineItemRepo
	outbox *planOutbox
	uc     *appprocurement.CreatePurchaseOrderFromPlanUseCase
}

func newPlanOrderFixture(t *testing.T, suppliers *stubSupplierFinder) planOrderFixture {
	pool := "pool"
	catalog := &stubCatalogReader{skus: []planning.PlanningSKU{
		{SKUID: "panettone", ProductName: "Panettone", Active: true},
		{SKUID: "colomba", ProductName: "Colomba", Active: true},
		{SKUID: "pool", ProductName: "Carne seca", IsWeightPool: true, Active: true},
		{SKUID: "pkg", PortionedFromSKUID: &pool, Active: true},
		{SKUID: "old", Active: false},
	}}
	supply := &stubSupplyReader{sourcing: map[string]planning.Sourcing{
		"panettone": {SupplierID: "sup-1", UnitsPerPurchasedItem: 6, UnitCostMinorUnits: 450, CurrencyCode: "GBP"},
		"colomba":   {SupplierID: "other", UnitsPerPurchasedItem: 12, UnitCostMinorUnits: 999, CurrencyCode: "BRL"},
	}}
	f := planOrderFixture{po: &stubPurchaseOrderRepo{}, lines: &stubLineItemRepo{}, outbox: &planOutbox{}}
	f.uc = appprocurement.NewCreatePurchaseOrderFromPlanUseCase(newMockGormDB(t), appprocurement.PlanOrderDeps{
		Catalog: catalog, Supply: supply, Suppliers: suppliers, Settings: &stubTenantSettings{currency: "EUR"},
		PurchaseOrders: f.po, LineItems: f.lines, Outbox: f.outbox,
	})
	return f
}

func TestCreatePurchaseOrderFromPlan_DraftWithSupplierPricesAndPackSizes(t *testing.T) {
	f := newPlanOrderFixture(t, &stubSupplierFinder{})

	po, err := f.uc.Execute(context.Background(), appprocurement.PlanOrderInput{
		SupplierID: "sup-1",
		Lines:      []appprocurement.PlanOrderLine{{SKUID: "panettone", Quantity: 172}, {SKUID: "colomba", Quantity: 3}, {SKUID: "pool", Quantity: 42}},
	})

	require.NoError(t, err)
	assert.Equal(t, "po-1", po.ID)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusDraft, po.Status)
	assert.Equal(t, "sup-1", *po.SupplierID)
	assert.Equal(t, "GBP", po.CurrencyCode, "this supplier's last PO currency")

	require.Len(t, f.lines.created, 3)
	pan := f.lines.created[0]
	assert.Equal(t, "po-1", pan.PurchaseOrderID)
	assert.Equal(t, "panettone", *pan.SKUID)
	assert.Equal(t, "Panettone", pan.RawDescription)
	assert.Equal(t, int64(172), pan.Quantity)
	assert.Equal(t, int64(6), pan.UnitsPerPurchasedItem)
	assert.Equal(t, int64(450), pan.UnitCostMinorUnits)
	assert.Equal(t, procurementDomain.LineItemTypeProduct, pan.LineType)
	assert.True(t, pan.LineTypeConfirmed)
	assert.Equal(t, int64(0), pan.Position)

	// Last bought elsewhere: that supplier's price and pack don't apply.
	col := f.lines.created[1]
	assert.Equal(t, int64(1), col.UnitsPerPurchasedItem)
	assert.Equal(t, int64(0), col.UnitCostMinorUnits)
	assert.Equal(t, int64(1), col.Position)

	// A weight pool is bought per kg.
	assert.Equal(t, int64(1000), f.lines.created[2].UnitsPerPurchasedItem)

	require.Equal(t, []string{"procurement.purchase_order.created"}, f.outbox.subjects)
	var event map[string]any
	require.NoError(t, json.Unmarshal(f.outbox.payloads[0], &event))
	assert.Equal(t, "po-1", event["purchase_order_id"])
	assert.Equal(t, "sup-1", event["supplier_id"])
	assert.Equal(t, "GBP", event["currency_code"])
	assert.Equal(t, "purchase_planner", event["source"])
	assert.Equal(t, float64(3), event["line_count"])
	assert.NotEmpty(t, event["event_id"])
}

func TestCreatePurchaseOrderFromPlan_OutboxFailureFailsTheWhole(t *testing.T) {
	f := newPlanOrderFixture(t, &stubSupplierFinder{})
	boom := errors.New("outbox down")
	f.outbox.err = boom

	_, err := f.uc.Execute(context.Background(), appprocurement.PlanOrderInput{
		SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "panettone", Quantity: 1}},
	})

	assert.ErrorIs(t, err, boom)
}

func TestCreatePurchaseOrderFromPlan_FallsBackToTenantCurrency(t *testing.T) {
	f := newPlanOrderFixture(t, &stubSupplierFinder{})

	po, err := f.uc.Execute(context.Background(), appprocurement.PlanOrderInput{
		SupplierID: "new-supplier", Lines: []appprocurement.PlanOrderLine{{SKUID: "colomba", Quantity: 1}},
	})

	require.NoError(t, err)
	assert.Equal(t, "EUR", po.CurrencyCode)
}

func TestCreatePurchaseOrderFromPlan_RejectsInvalidOrders(t *testing.T) {
	cases := map[string]struct {
		in   appprocurement.PlanOrderInput
		want error
	}{
		"no lines":     {appprocurement.PlanOrderInput{SupplierID: "sup-1"}, appprocurement.ErrInvalidPlanOrder},
		"zero qty":     {appprocurement.PlanOrderInput{SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "panettone"}}}, appprocurement.ErrInvalidPlanOrder},
		"duplicate":    {appprocurement.PlanOrderInput{SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "panettone", Quantity: 1}, {SKUID: "panettone", Quantity: 2}}}, appprocurement.ErrInvalidPlanOrder},
		"unknown sku":  {appprocurement.PlanOrderInput{SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "nope", Quantity: 1}}}, appprocurement.ErrSKUNotPlannable},
		"package sku":  {appprocurement.PlanOrderInput{SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "pkg", Quantity: 1}}}, appprocurement.ErrSKUNotPlannable},
		"inactive sku": {appprocurement.PlanOrderInput{SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "old", Quantity: 1}}}, appprocurement.ErrSKUNotPlannable},
	}
	for name, c := range cases {
		f := newPlanOrderFixture(t, &stubSupplierFinder{})
		_, err := f.uc.Execute(context.Background(), c.in)
		assert.ErrorIs(t, err, c.want, name)
		assert.False(t, f.po.createCalled, name)
	}
}

func TestCreatePurchaseOrderFromPlan_UnknownSupplier(t *testing.T) {
	f := newPlanOrderFixture(t, &stubSupplierFinder{err: suppliersDomain.ErrSupplierNotFound})

	_, err := f.uc.Execute(context.Background(), appprocurement.PlanOrderInput{
		SupplierID: "ghost", Lines: []appprocurement.PlanOrderLine{{SKUID: "panettone", Quantity: 1}},
	})

	assert.ErrorIs(t, err, suppliersDomain.ErrSupplierNotFound)
	assert.False(t, f.po.createCalled)
}

func TestCreatePurchaseOrderFromPlan_LineFailureFailsTheWhole(t *testing.T) {
	f := newPlanOrderFixture(t, &stubSupplierFinder{})
	boom := errors.New("boom")
	f.lines.createErr = boom

	_, err := f.uc.Execute(context.Background(), appprocurement.PlanOrderInput{
		SupplierID: "sup-1", Lines: []appprocurement.PlanOrderLine{{SKUID: "panettone", Quantity: 1}},
	})

	assert.ErrorIs(t, err, boom)
}
