package main

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/storage"
	"github.com/adirkuhn/commerceos/internal/config"
)

// costLedgerRecorderAdapter bridges application/inventory's local
// costLedgerRecorder interface to application/pms's real
// RecordCostLedgerMovementUseCase -- lives here (not in either application
// package) because application/pms already imports application/inventory
// (see application/pms/recalculate_cost_prices.go's doc comment), so the
// reverse import would cycle.
type costLedgerRecorderAdapter struct {
	uc *pmsapp.RecordCostLedgerMovementUseCase
}

func (a costLedgerRecorderAdapter) Execute(ctx context.Context, in invapp.CostLedgerMovementInput) error {
	return a.uc.Execute(ctx, pmsapp.RecordCostLedgerMovementInput{
		SKUID: in.SKUID, QuantityDelta: in.QuantityDelta,
		SourceType:    pmsDomain.SKUCostPriceLedgerSourceType(in.SourceType),
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

// buildFileStorage mirrors cmd/api/main.go's helper of the same name —
// each binary wires its own dependencies (no shared main-package library
// between commands), matching this codebase's existing convention.
func buildFileStorage(cfg config.Config) pmsDomain.FileStorage {
	switch cfg.StorageDriver {
	case "s3":
		if cfg.S3Endpoint == "" || cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" || cfg.S3Bucket == "" || cfg.S3PublicBaseURL == "" {
			log.Fatalf("storage: STORAGE_DRIVER=s3 requires S3_ENDPOINT, S3_ACCESS_KEY_ID, S3_SECRET_ACCESS_KEY, S3_BUCKET, S3_PUBLIC_BASE_URL")
		}
		client, err := minio.New(cfg.S3Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
			Secure: cfg.S3UseSSL,
		})
		if err != nil {
			log.Fatalf("storage: minio client: %v", err)
		}
		return storage.NewS3Storage(client, cfg.S3Bucket, cfg.S3PublicBaseURL)
	case "local":
		return storage.NewLocalStorage(cfg.MediaUploadDir, cfg.PublicAPIBaseURL+"/api/v1/uploads")
	default:
		log.Fatalf("storage: unknown STORAGE_DRIVER %q (want \"local\" or \"s3\")", cfg.StorageDriver)
		return nil
	}
}
