package oms

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type RecordSalesOrderActivityInput struct {
	SalesOrderID string
	UserID       string
	ActivityType string
	Summary      string
	// Metadata is arbitrary JSON-able detail (e.g. a MyParcel API
	// response) -- nil is recorded as an empty object, never a JSON null,
	// so callers never need to special-case reading it back.
	Metadata map[string]any
}

// RecordSalesOrderActivityUseCase is the single write path for the order
// activity log (see domain/oms.SalesOrderActivity's doc) -- shipping's use
// cases depend on this via interface rather than touching
// domain/oms.SalesOrderActivityRepository directly, per CLAUDE.md's
// cross-domain-calls-through-use-cases rule.
type RecordSalesOrderActivityUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	activityRepo   omsDomain.SalesOrderActivityRepository
}

func NewRecordSalesOrderActivityUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, activityRepo omsDomain.SalesOrderActivityRepository, outbox eventrecord.Enqueuer) *RecordSalesOrderActivityUseCase {
	return &RecordSalesOrderActivityUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, activityRepo: activityRepo}
}

func (uc *RecordSalesOrderActivityUseCase) Execute(ctx context.Context, in RecordSalesOrderActivityInput) (*omsDomain.SalesOrderActivity, error) {
	metadata := in.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("recordSalesOrderActivity: marshal metadata: %w", err)
	}

	activity := &omsDomain.SalesOrderActivity{
		SalesOrderID: in.SalesOrderID, UserID: in.UserID,
		ActivityType: in.ActivityType, Summary: in.Summary, Metadata: metadataJSON,
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.salesOrderRepo.FindByID(ctx, in.SalesOrderID); err != nil {
			return err
		}
		activity.TenantID = tenant.IDFromContext(ctx)
		if err := uc.activityRepo.Create(ctx, activity); err != nil {
			return err
		}
		return recordActivity(ctx, uc.outbox, *activity)
	})
	if err != nil {
		return nil, fmt.Errorf("recordSalesOrderActivity: %w", err)
	}
	return activity, nil
}
