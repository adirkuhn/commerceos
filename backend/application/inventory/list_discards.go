package inventory

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	defaultDiscardsPerPage = 50
	maxDiscardsPerPage     = 200
)

type ListDiscardsInput struct {
	WarehouseID *string
	SKUID       *string
	Type        *invDomain.MovementType
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
}

type ListDiscardsResult struct {
	Discards []invDomain.DiscardRecordReport
	Total    int64
}

// ListDiscardsUseCase is the discarded-items report's data source --
// tenant-wide, newest first, optionally narrowed by warehouse/SKU/date
// range.
type ListDiscardsUseCase struct {
	db          *gorm.DB
	discardRepo invDomain.DiscardRecordRepository
}

func NewListDiscardsUseCase(db *gorm.DB, discardRepo invDomain.DiscardRecordRepository) *ListDiscardsUseCase {
	return &ListDiscardsUseCase{db: db, discardRepo: discardRepo}
}

func (uc *ListDiscardsUseCase) Execute(ctx context.Context, in ListDiscardsInput) (*ListDiscardsResult, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = defaultDiscardsPerPage
	}
	if limit > maxDiscardsPerPage {
		limit = maxDiscardsPerPage
	}

	result := &ListDiscardsResult{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		discards, total, err := uc.discardRepo.List(ctx, invDomain.DiscardRecordFilter{
			WarehouseID: in.WarehouseID, SKUID: in.SKUID, Type: in.Type, From: in.From, To: in.To, Limit: limit, Offset: in.Offset,
		})
		if err != nil {
			return err
		}
		result.Discards = discards
		result.Total = total
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listDiscards: %w", err)
	}
	return result, nil
}
