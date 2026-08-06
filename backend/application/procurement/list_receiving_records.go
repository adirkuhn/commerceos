package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ListReceivingRecordsUseCase struct {
	db      *gorm.DB
	recRepo procurementDomain.ReceivingRecordRepository
}

func NewListReceivingRecordsUseCase(db *gorm.DB, recRepo procurementDomain.ReceivingRecordRepository) *ListReceivingRecordsUseCase {
	return &ListReceivingRecordsUseCase{db: db, recRepo: recRepo}
}

func (uc *ListReceivingRecordsUseCase) Execute(ctx context.Context, purchaseOrderID string) ([]procurementDomain.ReceivingRecord, error) {
	var records []procurementDomain.ReceivingRecord
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		records, err = uc.recRepo.ListByPurchaseOrder(ctx, purchaseOrderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listReceivingRecords: %w", err)
	}
	return records, nil
}
