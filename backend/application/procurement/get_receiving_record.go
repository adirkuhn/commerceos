package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ReceivingRecordDetail struct {
	ReceivingRecord procurementDomain.ReceivingRecord
	Discrepancies   []procurementDomain.ReceivingDiscrepancy
}

// GetReceivingRecordUseCase mirrors GetPurchaseOrderUseCase's shape:
// one aggregate row plus its child rows for the review UI.
type GetReceivingRecordUseCase struct {
	db              *gorm.DB
	recRepo         procurementDomain.ReceivingRecordRepository
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository
}

func NewGetReceivingRecordUseCase(
	db *gorm.DB,
	recRepo procurementDomain.ReceivingRecordRepository,
	discrepancyRepo procurementDomain.ReceivingDiscrepancyRepository,
) *GetReceivingRecordUseCase {
	return &GetReceivingRecordUseCase{db: db, recRepo: recRepo, discrepancyRepo: discrepancyRepo}
}

func (uc *GetReceivingRecordUseCase) Execute(ctx context.Context, id string) (*ReceivingRecordDetail, error) {
	var detail ReceivingRecordDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		rec, err := uc.recRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		detail.ReceivingRecord = *rec

		discrepancies, err := uc.discrepancyRepo.ListByReceivingRecord(ctx, id)
		if err != nil {
			return err
		}
		detail.Discrepancies = discrepancies
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getReceivingRecord: %w", err)
	}
	return &detail, nil
}
