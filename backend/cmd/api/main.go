package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	genia "github.com/adirkuhn/commerceos/api/iam"
	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	gennotifications "github.com/adirkuhn/commerceos/api/notifications"
	genpms "github.com/adirkuhn/commerceos/api/pms"
	gensettings "github.com/adirkuhn/commerceos/api/settings"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/iam"
	inventoryhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/inventory"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
	notificationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/notifications"
	pmshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pms"
	settingshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/settings"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/storage"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
)

// buildFileStorage selects the pms.FileStorage adapter per cfg.StorageDriver.
// Kept out of infrastructure/handlers/pms so that package stays storage-
// implementation-agnostic (it only depends on the domain port).
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

func main() {
	cfg := config.Load()

	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg.FrontendBaseURL))
	r.Use(middleware.ErrorHandler())

	v1 := r.Group("/api/v1")
	v1.Use(middleware.JWT([]byte(cfg.JWTSecret)))

	iamSpec, err := genia.GetSpec()
	if err != nil {
		log.Fatalf("iam: load openapi spec: %v", err)
	}
	// Own sub-group (see pmsGroup below) so the iam-only spec validator
	// never applies to integrations/pms routes.
	iamGroup := v1.Group("")
	iamGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(iamSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	iam.Register(iamGroup, db)

	integrationsSpec, err := genintegrations.GetSpec()
	if err != nil {
		log.Fatalf("integrations: load openapi spec: %v", err)
	}
	// Own sub-group (see pmsGroup below) so the integrations-only spec
	// validator never applies to iam/pms routes.
	integrationsGroup := v1.Group("")
	integrationsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(integrationsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))

	outboxRepo := repositories.NewOutboxRepository(db)
	wooFactory := woocommerce.NewFactory()
	integrations.Register(integrationsGroup, r, db, outboxRepo, wooFactory, []byte(cfg.JWTSecret),
		cfg.IntegrationsEncryptionSecret, cfg.PublicAPIBaseURL, cfg.FrontendBaseURL)

	pmsSpec, err := genpms.GetSpec()
	if err != nil {
		log.Fatalf("pms: load openapi spec: %v", err)
	}
	// Own sub-group so the PMS-only spec validator never applies to
	// iam/integrations routes — scoping via v1.Use() directly would silently
	// depend on registration order (only routes registered *after* the Use
	// call would be affected), which is fragile and non-obvious to a future
	// reader reordering these calls.
	pmsGroup := v1.Group("")
	pmsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(pmsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	fileStorage := buildFileStorage(cfg)
	pmshandler.Register(pmsGroup, db, fileStorage, outboxRepo)

	inventorySpec, err := geninventory.GetSpec()
	if err != nil {
		log.Fatalf("inventory: load openapi spec: %v", err)
	}
	// Own sub-group so the inventory-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	inventoryGroup := v1.Group("")
	inventoryGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(inventorySpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	inventoryhandler.Register(inventoryGroup, db)

	settingsSpec, err := gensettings.GetSpec()
	if err != nil {
		log.Fatalf("settings: load openapi spec: %v", err)
	}
	// Own sub-group so the settings-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	settingsGroup := v1.Group("")
	settingsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(settingsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	settingshandler.Register(settingsGroup, db)

	notificationsSpec, err := gennotifications.GetSpec()
	if err != nil {
		log.Fatalf("notifications: load openapi spec: %v", err)
	}
	// Own sub-group so the notifications-only spec validator never applies
	// to the other modules' routes — same reasoning as pmsGroup above.
	notificationsGroup := v1.Group("")
	notificationsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(notificationsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	notificationshandler.Register(notificationsGroup, db)

	// Uploaded product images — public, unauthenticated (product images are
	// meant to be publicly viewable, same as any storefront/CDN asset).
	// Mounted on the root router, not v1 (which has JWT middleware), and
	// matches the URL infrastructure/storage.LocalStorage builds. Only
	// needed for the local driver — S3/MinIO serves objects directly from
	// the bucket at S3PublicBaseURL.
	if cfg.StorageDriver == "local" {
		r.Static("/api/v1/uploads", cfg.MediaUploadDir)
	}

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("api listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("run: %v", err)
	}
}
