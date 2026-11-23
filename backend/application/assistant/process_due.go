package assistant

//events:none claiming a queued request and requeueing a stale one is queue bookkeeping, not a business action -- the attempt's outcome raises the event (process.go)

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// requestProcessor is ProcessRequestUseCase narrowed to point of use.
type requestProcessor interface {
	Execute(ctx context.Context, r assistant.Request) error
}

// ProcessDueRequestsUseCase is the worker tick: claim the due queued
// requests across every tenant, then run each one as the person who asked.
type ProcessDueRequestsUseCase struct {
	db        *gorm.DB
	requests  assistant.RequestRepository
	processor requestProcessor
	now       func() time.Time
	batchSize int
}

func NewProcessDueRequestsUseCase(db *gorm.DB, requests assistant.RequestRepository, processor requestProcessor, now func() time.Time, batchSize int) *ProcessDueRequestsUseCase {
	return &ProcessDueRequestsUseCase{db: db, requests: requests, processor: processor, now: now, batchSize: batchSize}
}

// Execute returns how many requests it ran. One request failing is logged
// and never stops the others.
func (uc *ProcessDueRequestsUseCase) Execute(ctx context.Context) (int, error) {
	var claimed []assistant.Request
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		at := uc.now()
		due, err := uc.requests.LockDue(ctx, at, uc.batchSize)
		if err != nil {
			return err
		}
		for i := range due {
			if err := due[i].Start(at); err != nil {
				return err
			}
			if err := uc.requests.Update(ctx, &due[i]); err != nil {
				return err
			}
		}
		claimed = due
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("processDueAssistantRequests: claim: %w", err)
	}
	for _, r := range claimed {
		if err := uc.processor.Execute(asRequester(ctx, r), r); err != nil {
			log.Printf("assistant request %s: %v", r.ID, err)
		}
	}
	return len(claimed), nil
}

// asRequester scopes ctx to the request's tenant, user and role -- tools
// then see exactly what the person who asked can see.
func asRequester(ctx context.Context, r assistant.Request) context.Context {
	ctx = tenant.AsTenant(ctx, r.TenantID)
	ctx = tenant.WithUserID(ctx, r.UserID)
	return tenant.WithRole(ctx, tenant.Role(r.UserRole))
}

// RequeueStaleRequestsUseCase puts requests left in processing by a worker
// that died mid-attempt back in the queue (run at scheduler start-up).
type RequeueStaleRequestsUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
	now      func() time.Time
}

func NewRequeueStaleRequestsUseCase(db *gorm.DB, requests assistant.RequestRepository, now func() time.Time) *RequeueStaleRequestsUseCase {
	return &RequeueStaleRequestsUseCase{db: db, requests: requests, now: now}
}

func (uc *RequeueStaleRequestsUseCase) Execute(ctx context.Context, olderThan time.Duration) (int64, error) {
	var n int64
	err := tenant.WithTenant(tenant.WithSuperAdmin(ctx), uc.db, func(ctx context.Context) error {
		at := uc.now()
		var err error
		n, err = uc.requests.RequeueStale(ctx, at.Add(-olderThan), at)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("requeueStaleAssistantRequests: %w", err)
	}
	return n, nil
}
