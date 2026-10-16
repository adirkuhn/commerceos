package procurement

import (
	"context"
	"fmt"
	"unicode/utf8"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// maxSupplierIssueNoteLength bounds the free-text note on a claim (e.g. a
// credit note number).
const maxSupplierIssueNoteLength = 1000

type UpdateSupplierIssueStatusInput struct {
	PurchaseOrderID string
	Ref             procurementDomain.SupplierIssueRef
	Status          procurementDomain.SupplierIssueStatus
	Note            string
}

// UpdateSupplierIssueStatusUseCase moves one supplier issue between open,
// claimed and resolved, with an optional note.
type UpdateSupplierIssueStatusUseCase struct {
	db   *gorm.DB
	deps SupplierIssueDeps
}

func NewUpdateSupplierIssueStatusUseCase(db *gorm.DB, deps SupplierIssueDeps) *UpdateSupplierIssueStatusUseCase {
	return &UpdateSupplierIssueStatusUseCase{db: db, deps: deps}
}

func (uc *UpdateSupplierIssueStatusUseCase) Execute(ctx context.Context, in UpdateSupplierIssueStatusInput) (*SupplierIssueView, error) {
	if !in.Status.Valid() {
		return nil, fmt.Errorf("updateSupplierIssueStatus: %w", procurementDomain.ErrInvalidSupplierIssueStatus)
	}
	if utf8.RuneCountInString(in.Note) > maxSupplierIssueNoteLength {
		return nil, fmt.Errorf("updateSupplierIssueStatus: %w", procurementDomain.ErrSupplierIssueNoteTooLong)
	}
	if !in.Ref.Source.Valid() {
		return nil, fmt.Errorf("updateSupplierIssueStatus: %w", procurementDomain.ErrSupplierIssueNotFound)
	}
	var view *SupplierIssueView
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		list, err := uc.deps.list(ctx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		for _, v := range list.Issues {
			if v.Ref != in.Ref {
				continue
			}
			updated, err := uc.deps.setStatus(ctx, list.PurchaseOrderID, v.SupplierIssue, in.Status, in.Note)
			if err != nil {
				return err
			}
			view = &SupplierIssueView{SupplierIssue: updated, SKUCode: v.SKUCode, ProductName: v.ProductName}
			return nil
		}
		return procurementDomain.ErrSupplierIssueNotFound
	})
	if err != nil {
		return nil, fmt.Errorf("updateSupplierIssueStatus: %w", err)
	}
	return view, nil
}
