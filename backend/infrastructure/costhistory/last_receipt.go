// Package costhistory wires the supplier cost history (procurement) into
// pms's cost price (COM-11). It lives in infrastructure because
// application/procurement imports application/pms, so pms can only reach
// procurement through a port adapted here.
package costhistory

import (
	"context"

	"gorm.io/gorm"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// summaryReader is application/procurement.GetSupplierCostSummaryUseCase (ISP).
type summaryReader interface {
	Execute(ctx context.Context, skuID string) (appprocurement.SupplierCostReport, error)
}

// LastReceipt answers pms's "what did the last receipt cost" from the
// supplier cost history summary.
type LastReceipt struct{ summaries summaryReader }

func NewLastReceipt(summaries summaryReader) LastReceipt { return LastReceipt{summaries: summaries} }

// NewLastReceiptFromDB builds LastReceipt over the real repositories.
func NewLastReceiptFromDB(db *gorm.DB) LastReceipt {
	return NewLastReceipt(appprocurement.NewGetSupplierCostSummaryUseCase(db,
		repositories.NewSupplierCostRepo(db), repositories.NewTenantSettingsRepo(db)))
}

// LastReceiptCost is nil when the SKU has no paid receipt standing.
func (l LastReceipt) LastReceiptCost(ctx context.Context, skuID string) (*apppms.LastReceiptCost, error) {
	report, err := l.summaries.Execute(ctx, skuID)
	if err != nil {
		return nil, err
	}
	last := report.Summary.Last
	if last == nil {
		return nil, nil
	}
	return &apppms.LastReceiptCost{
		LandedUnitCostMinorUnits: last.LandedUnitCostMinorUnits,
		PurchaseOrderLineItemID:  last.PurchaseOrderLineItemID,
	}, nil
}
