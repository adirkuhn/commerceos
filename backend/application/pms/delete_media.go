package pms

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteMediaUseCase struct {
	db        *gorm.DB
	mediaRepo pmsDomain.MediaRepository
	storage   pmsDomain.FileStorage
}

func NewDeleteMediaUseCase(db *gorm.DB, mediaRepo pmsDomain.MediaRepository, storage pmsDomain.FileStorage) *DeleteMediaUseCase {
	return &DeleteMediaUseCase{db: db, mediaRepo: mediaRepo, storage: storage}
}

// Execute deletes the DB row first, then the stored file. That order means
// a failure never leaves a Media row pointing at a file that's already
// gone (a broken image); worst case on a storage failure is an orphaned
// file using disk space, which is the safer side to fail on.
func (uc *DeleteMediaUseCase) Execute(ctx context.Context, id string) error {
	var url string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		m, err := uc.mediaRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		url = m.URL
		return uc.mediaRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteMedia: %w", err)
	}

	if err := uc.storage.Delete(ctx, url); err != nil {
		return fmt.Errorf("deleteMedia: %w", err)
	}
	return nil
}
