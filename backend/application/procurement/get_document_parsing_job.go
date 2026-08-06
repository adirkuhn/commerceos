package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// GetDocumentParsingJobUseCase lets the review UI poll a job's status after
// AttachDocumentToPurchaseOrderUseCase/CreatePurchaseOrderFromDocumentUseCase
// enqueue it -- cmd/scheduler writes ErrorMessage on failure, this is what
// makes that message reachable by the client instead of stuck in the DB.
type GetDocumentParsingJobUseCase struct {
	db      *gorm.DB
	jobRepo procurementDomain.DocumentParsingJobRepository
}

func NewGetDocumentParsingJobUseCase(db *gorm.DB, jobRepo procurementDomain.DocumentParsingJobRepository) *GetDocumentParsingJobUseCase {
	return &GetDocumentParsingJobUseCase{db: db, jobRepo: jobRepo}
}

func (uc *GetDocumentParsingJobUseCase) Execute(ctx context.Context, id string) (*procurementDomain.DocumentParsingJob, error) {
	var job *procurementDomain.DocumentParsingJob
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		j, err := uc.jobRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		job = j
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getDocumentParsingJob: %w", err)
	}
	return job, nil
}
