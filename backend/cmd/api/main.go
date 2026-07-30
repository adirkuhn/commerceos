package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	genia "github.com/adirkuhn/commerceos/api/iam"
	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	genpms "github.com/adirkuhn/commerceos/api/pms"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/iam"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
	pmshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
)

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
	pmshandler.Register(pmsGroup, db)

	addr := fmt.Sprintf(":%s", cfg.HTTPPort)
	log.Printf("api listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("run: %v", err)
	}
}
