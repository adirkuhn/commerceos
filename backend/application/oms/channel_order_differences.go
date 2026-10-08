package oms

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RecordChannelOrderDifferencesInput is the channel's own view of an
// imported order: its lines, its total, and what those lines (with
// shipping and fees) add up to.
type RecordChannelOrderDifferencesInput struct {
	SalesOrderID                string
	ChannelLines                []omsDomain.ChannelLine
	ChannelTotalMinorUnits      int64
	ChannelLinesTotalMinorUnits int64
}

// checkedDiscrepancyKinds: one comparison covers every kind.
var checkedDiscrepancyKinds = []omsDomain.LineDiscrepancyKind{
	omsDomain.LineDiscrepancyMissing, omsDomain.LineDiscrepancyQuantity, omsDomain.LineDiscrepancyTotal,
}

// RecordChannelOrderDifferencesUseCase compares an imported order with the
// channel's own order and stores what differs (see domain/oms.LineDiscrepancy)
// for the Stock Issues page. It never changes the order: staff decide what
// to do. Called on every import of the order, so a difference that's since
// been fixed by hand resolves itself.
type RecordChannelOrderDifferencesUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	discrepancies  omsDomain.LineDiscrepancyRepository
	outbox         eventrecord.Enqueuer
	now            func() time.Time
}

func NewRecordChannelOrderDifferencesUseCase(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	discrepancies omsDomain.LineDiscrepancyRepository,
	outbox eventrecord.Enqueuer,
	now func() time.Time,
) *RecordChannelOrderDifferencesUseCase {
	return &RecordChannelOrderDifferencesUseCase{
		db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, discrepancies: discrepancies,
		outbox: outbox, now: now,
	}
}

func (uc *RecordChannelOrderDifferencesUseCase) Execute(ctx context.Context, in RecordChannelOrderDifferencesInput) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, in.SalesOrderID)
		if err != nil {
			return err
		}
		local, err := uc.lineItemRepo.ListBySalesOrder(ctx, order.ID)
		if err != nil {
			return err
		}
		existing, err := uc.discrepancies.ListBySalesOrder(ctx, order.ID)
		if err != nil {
			return err
		}

		found := omsDomain.CompareChannelLines(in.ChannelLines, local)
		if omsDomain.ChannelTotalsDiffer(in.ChannelTotalMinorUnits, in.ChannelLinesTotalMinorUnits) {
			found = append(found, omsDomain.TotalDiscrepancy(in.ChannelTotalMinorUnits, in.ChannelLinesTotalMinorUnits))
		}
		plan := omsDomain.ReconcileLineDiscrepancies(existing, found, checkedDiscrepancyKinds, uc.now())

		for _, d := range plan.Create {
			d.TenantID, d.SalesOrderID = order.TenantID, order.ID
			if err := uc.discrepancies.Create(ctx, &d); err != nil {
				return err
			}
			if err := recordDiscrepancy(ctx, uc.outbox, events.LineDiscrepancyDetected, d); err != nil {
				return err
			}
		}
		for _, d := range plan.Update {
			if err := uc.discrepancies.Save(ctx, &d); err != nil {
				return err
			}
			action := events.LineDiscrepancyDetected
			if d.ResolvedAt != nil {
				action = events.LineDiscrepancyResolved
			}
			if err := recordDiscrepancy(ctx, uc.outbox, action, d); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("recordChannelOrderDifferences: %w", err)
	}
	return nil
}

func recordDiscrepancy(ctx context.Context, outbox eventrecord.Enqueuer, action events.ActionType, d omsDomain.LineDiscrepancy) error {
	return eventrecord.Record(ctx, outbox, action, d.ID, eventrecord.Data{
		"sales_order_id": d.SalesOrderID, "kind": string(d.Kind), "sku_code": d.SKUCode, "name": d.Name,
		"channel_quantity": d.ChannelQuantity, "local_quantity": d.LocalQuantity,
		"channel_total_minor_units": d.ChannelTotalMinorUnits, "local_total_minor_units": d.LocalTotalMinorUnits,
	})
}

// DismissLineDiscrepancyUseCase: staff handled a reported order difference
// (or decided it needs nothing) -- it leaves the Stock Issues list and
// stays dismissed unless its numbers change.
type DismissLineDiscrepancyUseCase struct {
	db            *gorm.DB
	discrepancies omsDomain.LineDiscrepancyRepository
	outbox        eventrecord.Enqueuer
	now           func() time.Time
}

func NewDismissLineDiscrepancyUseCase(db *gorm.DB, discrepancies omsDomain.LineDiscrepancyRepository, outbox eventrecord.Enqueuer, now func() time.Time) *DismissLineDiscrepancyUseCase {
	return &DismissLineDiscrepancyUseCase{db: db, discrepancies: discrepancies, outbox: outbox, now: now}
}

func (uc *DismissLineDiscrepancyUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		d, err := uc.discrepancies.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if err := d.Dismiss(uc.now()); err != nil {
			return err
		}
		if err := uc.discrepancies.Save(ctx, d); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.LineDiscrepancyDismissed, d.ID, eventrecord.Data{
			"sales_order_id": d.SalesOrderID, "kind": string(d.Kind), "sku_code": d.SKUCode, "name": d.Name,
		})
	})
	if err != nil {
		return fmt.Errorf("dismissLineDiscrepancy: %w", err)
	}
	return nil
}

// ListOpenLineDiscrepanciesUseCase is the Stock Issues page's "Order
// differences" list, newest first.
type ListOpenLineDiscrepanciesUseCase struct {
	db            *gorm.DB
	discrepancies omsDomain.LineDiscrepancyRepository
}

func NewListOpenLineDiscrepanciesUseCase(db *gorm.DB, discrepancies omsDomain.LineDiscrepancyRepository) *ListOpenLineDiscrepanciesUseCase {
	return &ListOpenLineDiscrepanciesUseCase{db: db, discrepancies: discrepancies}
}

func (uc *ListOpenLineDiscrepanciesUseCase) Execute(ctx context.Context, page, perPage int) ([]omsDomain.OpenLineDiscrepancy, int64, error) {
	var out []omsDomain.OpenLineDiscrepancy
	var total int64
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, total, err = uc.discrepancies.ListOpen(ctx, page, perPage)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("listOpenLineDiscrepancies: %w", err)
	}
	return out, total, nil
}
