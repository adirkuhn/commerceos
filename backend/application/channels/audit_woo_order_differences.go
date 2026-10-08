package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// AuditWooOrderDifferencesBatchSize is how many orders one call checks --
// each is a Woo API round trip, so a call stays inside a request's time
// budget; the caller repeats with NextCursor until it comes back empty.
const AuditWooOrderDifferencesBatchSize = 25

type AuditWooOrderDifferencesInput struct {
	ConnectionID string
	// Cursor is the previous call's NextCursor -- empty starts from the
	// newest order.
	Cursor string
}

type AuditWooOrderDifferencesResult struct {
	Checked int
	// Errors names each order that couldn't be fetched or compared -- one
	// failure never stops the rest of the batch.
	Errors []string
	// NextCursor is empty once there is nothing left to check.
	NextCursor string
}

// AuditWooOrderDifferencesUseCase compares already-imported Woo orders
// with Woo's current order, newest first, and reports what differs on the
// Stock Issues page -- read-only towards both Woo and the orders. Orders
// that will never ship (cancelled, failed, refunded) are skipped.
type AuditWooOrderDifferencesUseCase struct {
	db          *gorm.DB
	orders      omsDomain.ChannelComparableOrderLister
	clients     wooClientResolver
	differences orderDifferenceRecorder
}

func NewAuditWooOrderDifferencesUseCase(db *gorm.DB, orders omsDomain.ChannelComparableOrderLister, clients wooClientResolver, differences orderDifferenceRecorder) *AuditWooOrderDifferencesUseCase {
	return &AuditWooOrderDifferencesUseCase{db: db, orders: orders, clients: clients, differences: differences}
}

func (uc *AuditWooOrderDifferencesUseCase) Execute(ctx context.Context, in AuditWooOrderDifferencesInput) (*AuditWooOrderDifferencesResult, error) {
	after, err := decodeBackfillCursor(in.Cursor)
	if err != nil {
		return nil, fmt.Errorf("auditWooOrderDifferences: %w", err)
	}
	client, err := uc.clients.ClientFor(ctx, in.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("auditWooOrderDifferences: %w", err)
	}

	var batch []omsDomain.SalesOrder
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		batch, err = uc.orders.ListChannelComparableOrders(ctx, in.ConnectionID, after, AuditWooOrderDifferencesBatchSize)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("auditWooOrderDifferences: %w", err)
	}

	result := &AuditWooOrderDifferencesResult{}
	for _, order := range batch {
		result.Checked++
		wo, err := client.GetOrder(ctx, order.ExternalOrderID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: fetch: %v", order.ExternalOrderID, err))
			continue
		}
		total, partsTotal := wooOrderTotals(*wo)
		if err := uc.differences.Execute(ctx, appoms.RecordChannelOrderDifferencesInput{
			SalesOrderID: order.ID, ChannelLines: wooChannelLines(*wo),
			ChannelTotalMinorUnits: total, ChannelLinesTotalMinorUnits: partsTotal,
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("woo order %s: compare: %v", order.ExternalOrderID, err))
		}
	}
	if len(batch) == AuditWooOrderDifferencesBatchSize {
		last := batch[len(batch)-1]
		result.NextCursor = encodeBackfillCursor(omsDomain.OrderPageCursor{OccurredAt: last.OccurredAt, ID: last.ID})
	}
	return result, nil
}
