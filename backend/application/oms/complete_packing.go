package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// MissingPackItem mirrors MissingPickItem -- one order line item that
// wasn't fully packed by the time the session was finished.
type MissingPackItem struct {
	LineItemID  string
	Name        string
	ExpectedQty int64
	PackedQty   int64
}

// CompletePackingResult: Completed false with StockShortfalls set means
// nothing was applied or completed yet -- staff must decide each shortfall
// (Resolutions) and finish again, same as picking used to.
type CompletePackingResult struct {
	Completed       bool
	Session         omsDomain.PackingSession
	MissingItems    []MissingPackItem
	StockShortfalls []StockShortfall
}

type CompletePackingInput struct {
	SessionID   string
	CompletedBy string
	// Resolutions decides, per pack scan ID, each stock shortfall a previous
	// attempt reported -- see StockShortfallResolution.
	Resolutions map[string]StockShortfallResolution
}

// CompletePackingUseCase is the packing screen's "Finish" action: applies
// every staged order change (substitutes, extra items, refunds -- see
// OrderChangeApplier) after a stock pre-check, then marks the session
// completed and the order packed and fulfilled (picking is optional). Lines
// not fully packed are reported (MissingItems), never blocking -- a manager
// may knowingly ship a partial box. Substituted and refunded units count as
// handled, except a substitute resolved "ship without".
type CompletePackingUseCase struct {
	db             *gorm.DB
	sessionRepo    omsDomain.PackingSessionRepository
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	scanRepo       omsDomain.PackScanRepository
	applier        *OrderChangeApplier
}

func NewCompletePackingUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	scanRepo omsDomain.PackScanRepository,
	applier *OrderChangeApplier,
) *CompletePackingUseCase {
	return &CompletePackingUseCase{
		db: db, sessionRepo: sessionRepo, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo, applier: applier,
	}
}

func (uc *CompletePackingUseCase) Execute(ctx context.Context, in CompletePackingInput) (*CompletePackingResult, error) {
	var session *omsDomain.PackingSession
	var order *omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	var scans []omsDomain.PackScan
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		session, err = uc.sessionRepo.FindByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PackingSessionStatusInProgress {
			return omsDomain.ErrPackingSessionNotActive
		}
		if session.Lock().IsHeldByOther(in.CompletedBy, time.Now()) {
			return omsDomain.ErrPackingSessionLocked
		}
		if order, err = uc.salesOrderRepo.FindByID(ctx, session.SalesOrderID); err != nil {
			return err
		}
		if lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, session.SalesOrderID); err != nil {
			return err
		}
		scans, err = uc.scanRepo.ListBySession(ctx, in.SessionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("completePacking: %w", err)
	}

	changes := packScanChanges(scans)
	shortfalls, err := uc.applier.CheckStockShortfalls(ctx, order.WarehouseID, changes, in.Resolutions)
	if err != nil {
		return nil, fmt.Errorf("completePacking: %w", err)
	}
	if len(shortfalls) > 0 {
		return &CompletePackingResult{StockShortfalls: shortfalls}, nil
	}
	// The applier commits each change in its own transaction (see
	// OrderChangeApplier.Apply), so it runs outside the one below.
	if err := uc.applier.Apply(ctx, "packing session "+in.SessionID, order, lineItems, changes, in.Resolutions, uc.scanRepo); err != nil {
		return nil, fmt.Errorf("completePacking: %w", err)
	}

	result := CompletePackingResult{Completed: true, MissingItems: missingPackItems(lineItems, scans, in.Resolutions)}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		session.Status = omsDomain.PackingSessionStatusCompleted
		session.CompletedAt = &now
		session.CompletedBy = in.CompletedBy
		session.ApplyLock(omsDomain.SessionLock{})
		if err := uc.sessionRepo.Update(ctx, session); err != nil {
			return err
		}
		if err := uc.sessionRepo.ReleaseLock(ctx, session.ID); err != nil {
			return err
		}
		if err := uc.salesOrderRepo.Update(ctx, session.SalesOrderID, map[string]any{"packed": true, "fulfilled": true}); err != nil {
			return err
		}
		result.Session = *session
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("completePacking: %w", err)
	}
	return &result, nil
}

// packScanChanges is every order change among a session's scans -- plain
// packs change nothing.
func packScanChanges(scans []omsDomain.PackScan) []omsDomain.StagedChange {
	var changes []omsDomain.StagedChange
	for _, s := range scans {
		if c, ok := s.Change(); ok {
			changes = append(changes, c)
		}
	}
	return changes
}

func missingPackItems(lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PackScan, resolutions map[string]StockShortfallResolution) []MissingPackItem {
	packedByLineItem := make(map[string]int64, len(scans))
	for _, s := range scans {
		// A substitute staff chose to ship without never replaced anything.
		if s.ActionType == omsDomain.PackScanActionSubstitute && resolutions[s.ID] == StockShortfallWithoutItem {
			continue
		}
		if id, qty := s.Covers(); id != "" {
			packedByLineItem[id] += qty
		}
	}
	var missing []MissingPackItem
	for _, li := range lineItems {
		packed := packedByLineItem[li.ID]
		if packed < li.Quantity {
			missing = append(missing, MissingPackItem{LineItemID: li.ID, Name: li.Name, ExpectedQty: li.Quantity, PackedQty: packed})
		}
	}
	return missing
}
