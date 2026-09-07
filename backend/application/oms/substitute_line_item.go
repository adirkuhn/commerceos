package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// SubstituteLineInput is one row of a substitute batch -- see
// SubstituteLineItemInput.
type SubstituteLineInput struct {
	SKUID              string
	SubstituteQuantity int64
	// OriginalQtyCovered is how much of the original line item's own
	// Quantity this row covers. Summed across every line in the same
	// batch, this must equal the original line's Quantity exactly.
	OriginalQtyCovered int64
	// ReturnToStock defaults false -- see PickScan.ReturnToStock's doc.
	ReturnToStock bool
}

type SubstituteLineItemInput struct {
	SessionID  string
	LineItemID string
	Lines      []SubstituteLineInput
	Reason     string
	ScannedBy  string
}

type SubstituteLineItemResult struct {
	Scans []omsDomain.PickScan
}

// SubstituteLineItemUseCase is the ONLY way a Substitute PickScan row gets
// created -- ScanPickItemUseCase explicitly refuses this action type (see
// ErrSubstituteViaBatchEndpointOnly). Stages one or more PickScan rows for
// the same original line item in a single call, one per substitute SKU,
// letting a single original unit be replaced by several different
// (typically cheaper) substitutes at once. Like ScanPickItemUseCase, this
// never touches real inventory or the order's line items -- CompletePickingUseCase's
// replay is still the only place any of it becomes real.
type SubstituteLineItemUseCase struct {
	db           *gorm.DB
	sessionRepo  omsDomain.PickingSessionRepository
	scanRepo     omsDomain.PickScanRepository
	lineItemRepo omsDomain.SalesOrderLineItemRepository
}

func NewSubstituteLineItemUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	scanRepo omsDomain.PickScanRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
) *SubstituteLineItemUseCase {
	return &SubstituteLineItemUseCase{db: db, sessionRepo: sessionRepo, scanRepo: scanRepo, lineItemRepo: lineItemRepo}
}

func (uc *SubstituteLineItemUseCase) Execute(ctx context.Context, in SubstituteLineItemInput) (*SubstituteLineItemResult, error) {
	if len(in.Lines) == 0 {
		return nil, fmt.Errorf("substituteLineItem: %w", omsDomain.ErrSubstituteLinesRequired)
	}
	var coverageSum int64
	for _, l := range in.Lines {
		if l.SKUID == "" || l.SubstituteQuantity <= 0 || l.OriginalQtyCovered <= 0 {
			return nil, fmt.Errorf("substituteLineItem: %w", omsDomain.ErrInvalidSubstituteQuantity)
		}
		coverageSum += l.OriginalQtyCovered
	}

	var scans []omsDomain.PickScan
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PickingSessionStatusInProgress {
			return omsDomain.ErrPickingSessionNotActive
		}

		li, err := uc.lineItemRepo.FindByID(ctx, in.LineItemID)
		if err != nil {
			return err
		}
		if li.SalesOrderID != session.SalesOrderID {
			return omsDomain.ErrLineItemNotOnOrder
		}
		if coverageSum != li.Quantity {
			return omsDomain.ErrSubstitutionCoverageMismatch
		}

		lineItemID := li.ID
		now := time.Now()
		scans = make([]omsDomain.PickScan, 0, len(in.Lines))
		for _, l := range in.Lines {
			covered := l.OriginalQtyCovered
			scan := omsDomain.PickScan{
				TenantID: tenant.IDFromContext(ctx), SessionID: in.SessionID, LineItemID: &lineItemID, SKUID: l.SKUID,
				ActionType: omsDomain.PickScanActionSubstitute, Quantity: l.SubstituteQuantity,
				OriginalQtyCovered: &covered, ReturnToStock: l.ReturnToStock, Reason: in.Reason,
				ScannedAt: now, ScannedBy: in.ScannedBy,
			}
			if err := uc.scanRepo.Create(ctx, &scan); err != nil {
				return err
			}
			scans = append(scans, scan)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("substituteLineItem: %w", err)
	}
	return &SubstituteLineItemResult{Scans: scans}, nil
}
