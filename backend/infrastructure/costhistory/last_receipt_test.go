package costhistory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/costhistory"
)

type stubSummaries struct {
	report appprocurement.SupplierCostReport
	err    error
}

func (s stubSummaries) Execute(context.Context, string) (appprocurement.SupplierCostReport, error) {
	return s.report, s.err
}

func TestLastReceipt(t *testing.T) {
	report := appprocurement.SupplierCostReport{Summary: procurementDomain.SupplierCostSummary{
		Last: &procurementDomain.LastSupplierCost{LandedUnitCostMinorUnits: 520, PurchaseOrderLineItemID: "line-1"},
	}}
	got, err := costhistory.NewLastReceipt(stubSummaries{report: report}).LastReceiptCost(context.Background(), "sku-1")
	require.NoError(t, err)
	assert.Equal(t, &apppms.LastReceiptCost{LandedUnitCostMinorUnits: 520, PurchaseOrderLineItemID: "line-1"}, got)

	got, err = costhistory.NewLastReceipt(stubSummaries{}).LastReceiptCost(context.Background(), "sku-1")
	require.NoError(t, err)
	assert.Nil(t, got, "no paid receipt")

	boom := errors.New("boom")
	_, err = costhistory.NewLastReceipt(stubSummaries{err: boom}).LastReceiptCost(context.Background(), "sku-1")
	assert.ErrorIs(t, err, boom)
}
