package inventory

import (
	"context"
	"fmt"
	"io"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

var allowedDiscardPhotoContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

type DiscardBatchInput struct {
	SKUID       string
	WarehouseID string
	// BatchID nil discards general-ledger (un-lotted) stock.
	BatchID     *string
	Quantity    int64 // always positive
	Reason      string
	DiscardedBy string
	OccurredAt  time.Time
	// Photo fields are all zero-value when no photo is attached.
	PhotoFilename    string
	PhotoContentType string
	PhotoData        io.Reader
}

// DiscardBatchUseCase is the general "throw this stock away" action --
// backs both the desktop expiry-review queue's Write-off (superseding a
// direct WriteOffExpiredBatchUseCase call there) and the scanner's Discard
// app. Unlike WriteOffExpiredBatchUseCase (which always empties a specific
// batch entirely), this takes an explicit quantity and works for a
// general-ledger row too -- the same shape Adjust/Move/Damaged-Lost already
// use elsewhere in this module.
type DiscardBatchUseCase struct {
	db                *gorm.DB
	batchRepo         invDomain.BatchRepository
	markDamagedOrLost damagedOrLostRecorder
	storage           invDomain.FileStorage
}

func NewDiscardBatchUseCase(
	db *gorm.DB,
	batchRepo invDomain.BatchRepository,
	markDamagedOrLost damagedOrLostRecorder,
	storage invDomain.FileStorage,
) *DiscardBatchUseCase {
	return &DiscardBatchUseCase{db: db, batchRepo: batchRepo, markDamagedOrLost: markDamagedOrLost, storage: storage}
}

// Execute uploads the optional photo (external I/O, outside any DB
// transaction -- same orphan-file-on-failure tradeoff as
// pms.UploadMediaUseCase), records a Lost movement for the given quantity
// (MarkDamagedOrLostUseCase writes the DiscardRecord itself -- see its own
// doc comment), then, for a real batch, marks it reviewed (same as
// WriteOffExpiredBatchUseCase's queue-clearing step).
func (uc *DiscardBatchUseCase) Execute(ctx context.Context, in DiscardBatchInput) (*invDomain.DiscardRecord, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("discardBatch: %w", invDomain.ErrInvalidQuantity)
	}
	if in.PhotoData != nil && !allowedDiscardPhotoContentTypes[in.PhotoContentType] {
		return nil, fmt.Errorf("discardBatch: %w", invDomain.ErrUnsupportedPhotoType)
	}

	var photoURL *string
	if in.PhotoData != nil {
		url, err := uc.storage.Save(ctx, tenant.IDFromContext(ctx), in.PhotoFilename, in.PhotoData)
		if err != nil {
			return nil, fmt.Errorf("discardBatch: %w", err)
		}
		photoURL = &url
	}

	occurredAt := in.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	_, record, err := uc.markDamagedOrLost.Execute(ctx, MarkDamagedOrLostInput{
		SKUID: in.SKUID, WarehouseID: in.WarehouseID, BatchID: in.BatchID,
		Quantity: in.Quantity, Type: invDomain.MovementTypeLost, Reason: in.Reason, OccurredAt: occurredAt,
		DiscardedBy: in.DiscardedBy, PhotoURL: photoURL,
	})
	if err != nil {
		return nil, fmt.Errorf("discardBatch: %w", err)
	}

	if in.BatchID != nil {
		err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			return uc.batchRepo.MarkReviewed(ctx, *in.BatchID)
		})
		if err != nil {
			return nil, fmt.Errorf("discardBatch: %w", err)
		}
	}
	return record, nil
}
