package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// GetReceivingParsingJobUseCase mirrors GetDocumentParsingJobUseCase for the
// invoice/receiving upload path.
type GetReceivingParsingJobUseCase struct {
	db      *gorm.DB
	jobRepo procurementDomain.ReceivingParsingJobRepository
}

func NewGetReceivingParsingJobUseCase(db *gorm.DB, jobRepo procurementDomain.ReceivingParsingJobRepository) *GetReceivingParsingJobUseCase {
	return &GetReceivingParsingJobUseCase{db: db, jobRepo: jobRepo}
}

func (uc *GetReceivingParsingJobUseCase) Execute(ctx context.Context, id string) (*procurementDomain.ReceivingParsingJob, error) {
	var job *procurementDomain.ReceivingParsingJob
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		j, err := uc.jobRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getReceivingParsingJob: %w", err)
	}
	return job, nil
}
