package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// supplierContacts is the supplier lookup the supplier issues list needs.
type supplierContacts interface {
	FindByID(ctx context.Context, id string) (*suppliersDomain.Supplier, error)
}

// SupplierIssueDeps are the ports shared by the supplier issue use cases
// (list, update status, email).
type SupplierIssueDeps struct {
	PurchaseOrders   procurementDomain.PurchaseOrderRepository
	Suppliers        supplierContacts
	LineItems        procurementDomain.PurchaseOrderLineItemRepository
	Bookings         procurementDomain.BookingRepository
	Scans            procurementDomain.BookingScanRepository
	ReceivingRecords procurementDomain.ReceivingRecordRepository
	Discrepancies    procurementDomain.ReceivingDiscrepancyRepository
	States           procurementDomain.SupplierIssueStateRepository
	SKUs             pmsDomain.SKURepository
	Variants         pmsDomain.VariantRepository
	Products         pmsDomain.ProductRepository
	Outbox           eventrecord.Enqueuer
	Now              func() time.Time
}

// SupplierContact is who the issues are claimed from.
type SupplierContact struct {
	ID    string
	Name  string
	Email string
}

// SupplierIssueView is a SupplierIssue with our own product names.
type SupplierIssueView struct {
	procurementDomain.SupplierIssue
	SKUCode     *string
	ProductName *string
}

type SupplierIssueList struct {
	PurchaseOrderID string
	// Supplier is nil when the PO has no supplier yet.
	Supplier *SupplierContact
	// BookingInProgress: a delivery is being booked right now -- its scans
	// show up once it is finished.
	BookingInProgress bool
	Issues            []SupplierIssueView
}

// ListSupplierIssuesUseCase is the "what to send back to the supplier"
// list of one purchase order: missing, damaged and lost items with their
// claim status.
type ListSupplierIssuesUseCase struct {
	db   *gorm.DB
	deps SupplierIssueDeps
}

func NewListSupplierIssuesUseCase(db *gorm.DB, deps SupplierIssueDeps) *ListSupplierIssuesUseCase {
	return &ListSupplierIssuesUseCase{db: db, deps: deps}
}

func (uc *ListSupplierIssuesUseCase) Execute(ctx context.Context, purchaseOrderID string) (*SupplierIssueList, error) {
	var list *SupplierIssueList
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		list, err = uc.deps.list(ctx, purchaseOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listSupplierIssues: %w", err)
	}
	return list, nil
}

// list builds the PO's issue list; call inside tenant.WithTenant.
func (d SupplierIssueDeps) list(ctx context.Context, purchaseOrderID string) (*SupplierIssueList, error) {
	po, err := d.PurchaseOrders.FindByID(ctx, purchaseOrderID)
	if err != nil {
		return nil, err
	}
	issues, err := d.collect(ctx, po.ID)
	if err != nil {
		return nil, err
	}
	active, err := d.Bookings.FindActiveByPurchaseOrder(ctx, po.ID)
	if err != nil {
		return nil, err
	}
	list := &SupplierIssueList{PurchaseOrderID: po.ID, BookingInProgress: active != nil, Issues: make([]SupplierIssueView, len(issues))}
	if po.SupplierID != nil {
		sup, err := d.Suppliers.FindByID(ctx, *po.SupplierID)
		if err != nil {
			return nil, err
		}
		list.Supplier = &SupplierContact{ID: sup.ID, Name: sup.Name, Email: sup.Email}
	}
	for i, issue := range issues {
		skuCode, productName, err := skuDisplayNames(ctx, d.SKUs, d.Variants, d.Products, issue.SKUID)
		if err != nil {
			return nil, err
		}
		list.Issues[i] = SupplierIssueView{SupplierIssue: issue, SKUCode: skuCode, ProductName: productName}
	}
	return list, nil
}

// collect derives the PO's issues; call inside tenant.WithTenant.
func (d SupplierIssueDeps) collect(ctx context.Context, purchaseOrderID string) ([]procurementDomain.SupplierIssue, error) {
	in := procurementDomain.SupplierIssueInputs{}
	var err error
	if in.LineItems, err = d.LineItems.ListByPurchaseOrder(ctx, purchaseOrderID); err != nil {
		return nil, err
	}
	bookings, err := d.Bookings.ListCompletedByPurchaseOrder(ctx, purchaseOrderID)
	if err != nil {
		return nil, err
	}
	for _, b := range bookings {
		scans, err := d.Scans.ListByBooking(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		in.Bookings = append(in.Bookings, procurementDomain.BookingWithScans{Booking: b, Scans: scans})
	}
	records, err := d.ReceivingRecords.ListByPurchaseOrder(ctx, purchaseOrderID)
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		discrepancies, err := d.Discrepancies.ListByReceivingRecord(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		in.ReceivingDiscrepancies = append(in.ReceivingDiscrepancies, discrepancies...)
	}
	if in.States, err = d.States.ListByPurchaseOrder(ctx, purchaseOrderID); err != nil {
		return nil, err
	}
	return procurementDomain.CollectSupplierIssues(in), nil
}

// setStatus stores issue's new status and records the change; a no-op when
// nothing changes. Returns the updated issue. Call inside tenant.WithTenant.
func (d SupplierIssueDeps) setStatus(ctx context.Context, purchaseOrderID string, issue procurementDomain.SupplierIssue,
	status procurementDomain.SupplierIssueStatus, note string) (procurementDomain.SupplierIssue, error) {
	if issue.Status == status && issue.StatusNote == note {
		return issue, nil
	}
	now := d.Now()
	st := procurementDomain.SupplierIssueState{
		TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: purchaseOrderID, Ref: issue.Ref,
		Status: status, Note: note, UpdatedBy: tenant.UserIDFromContext(ctx), UpdatedAt: now,
	}
	if err := d.States.Upsert(ctx, &st); err != nil {
		return issue, err
	}
	changes := eventrecord.NewChanges().Set("status", string(issue.Status), string(status)).Set("note", issue.StatusNote, note)
	if err := eventrecord.Record(ctx, d.Outbox, events.SupplierIssueUpdated, issue.Ref.SourceID, changes, eventrecord.Data{
		"purchase_order_id": purchaseOrderID, "source": string(issue.Ref.Source), "kind": string(issue.Kind),
	}); err != nil {
		return issue, err
	}
	issue.Status, issue.StatusNote, issue.StatusUpdatedAt = status, note, &now
	return issue, nil
}
