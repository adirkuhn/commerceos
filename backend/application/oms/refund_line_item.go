package oms

import (
	"context"
	"fmt"
	"time"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// customerUserRole mirrors middleware.RoleCustomerUser's string value --
// duplicated here rather than imported, since application/ must never
// depend on infrastructure/ (see CLAUDE.md's architecture rule). The value
// itself is a stable, documented part of the lm2 JWT role vocabulary
// (CLAUDE.md's Roles table), not a middleware implementation detail.
const customerUserRole = "customer_user"

// refundPermissionRule is the fine-grained RBAC rule (see iam.Rule's
// {system}.{resource}.{action} format) a customer_user needs to trigger a
// refund -- customer_owner/customer_admin/super_admin bypass this check
// entirely, same "full access to own tenant" convention as every other
// role-gated action in this codebase.
const refundPermissionRule = "oms.line_item.refund"

// RefundLineItemInput stages refunding part or all of an existing line
// item's own quantity -- see PickScanActionRefund's doc comment.
type RefundLineItemInput struct {
	SessionID  string
	LineItemID string
	// Quantity is the amount of the ORIGINAL line item being refunded --
	// positive, at most the line's own Quantity, need not equal it.
	Quantity int64
	Reason   string
	// ScannedBy is the PickScan row's own ScannedBy (lm2 user id) -- same
	// field every other picking action stamps.
	ScannedBy string
	// RequesterUserID/RequesterRole identify who's asking, for the
	// fine-grained RBAC check below -- a customer_owner/customer_admin/
	// super_admin always passes; a customer_user needs the
	// refundPermissionRule granted via their group.
	RequesterUserID string
	RequesterRole   string
}

type RefundLineItemResult struct {
	Scan omsDomain.PickScan
}

// RefundLineItemUseCase is the scanner app's "refund instead" action,
// alongside Pick/Substitute -- the only use case allowed to create a
// PickScanActionRefund row. Like every other picking action, this only
// stages the row: nothing is real (no money moves, no line item mutated)
// until CompletePickingUseCase replays it.
type RefundLineItemUseCase struct {
	db             *gorm.DB
	sessionRepo    omsDomain.PickingSessionRepository
	scanRepo       omsDomain.PickScanRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	permissionRepo iamDomain.GroupPermissionRepository
}

func NewRefundLineItemUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	scanRepo omsDomain.PickScanRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	permissionRepo iamDomain.GroupPermissionRepository,
) *RefundLineItemUseCase {
	return &RefundLineItemUseCase{
		db: db, sessionRepo: sessionRepo, scanRepo: scanRepo, lineItemRepo: lineItemRepo, permissionRepo: permissionRepo,
	}
}

func (uc *RefundLineItemUseCase) Execute(ctx context.Context, in RefundLineItemInput) (*RefundLineItemResult, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("refundLineItem: %w", omsDomain.ErrInvalidRefundQuantity)
	}

	var scan omsDomain.PickScan
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if in.RequesterRole == customerUserRole {
			allowed, err := uc.permissionRepo.HasPermission(ctx, in.RequesterUserID, iamDomain.Rule(refundPermissionRule))
			if err != nil {
				return err
			}
			if !allowed {
				return omsDomain.ErrPermissionDenied
			}
		}

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
		if li.SKUID == nil {
			return omsDomain.ErrRefundRequiresLinkedSKU
		}
		if in.Quantity > li.Quantity {
			return omsDomain.ErrInvalidRefundQuantity
		}

		lineItemID := li.ID
		scan = omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: in.SessionID, LineItemID: &lineItemID, SKUID: *li.SKUID,
			ActionType: omsDomain.PickScanActionRefund, Quantity: in.Quantity, Reason: in.Reason,
			ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
		}
		return uc.scanRepo.Create(ctx, &scan)
	})
	if err != nil {
		return nil, fmt.Errorf("refundLineItem: %w", err)
	}
	return &RefundLineItemResult{Scan: scan}, nil
}
