package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	genbolconfirmations "github.com/adirkuhn/commerceos/api/bolconfirmations"
	gencollections "github.com/adirkuhn/commerceos/api/collections"
	gencoupons "github.com/adirkuhn/commerceos/api/coupons"
	gencustomers "github.com/adirkuhn/commerceos/api/customers"
	gendashboard "github.com/adirkuhn/commerceos/api/dashboard"
	genemailintegration "github.com/adirkuhn/commerceos/api/emailintegration"
	genfx "github.com/adirkuhn/commerceos/api/fx"
	genia "github.com/adirkuhn/commerceos/api/iam"
	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	genmessagecenter "github.com/adirkuhn/commerceos/api/messagecenter"
	genmoneybird "github.com/adirkuhn/commerceos/api/moneybird"
	gennewsletter "github.com/adirkuhn/commerceos/api/newsletter"
	gennotifications "github.com/adirkuhn/commerceos/api/notifications"
	genoms "github.com/adirkuhn/commerceos/api/oms"
	genorderpayments "github.com/adirkuhn/commerceos/api/orderpayments"
	genpms "github.com/adirkuhn/commerceos/api/pms"
	genpos "github.com/adirkuhn/commerceos/api/pos"
	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	genpromotions "github.com/adirkuhn/commerceos/api/promotions"
	genpurchaseplanning "github.com/adirkuhn/commerceos/api/purchaseplanning"
	genquotes "github.com/adirkuhn/commerceos/api/quotes"
	genreports "github.com/adirkuhn/commerceos/api/reports"
	genreviews "github.com/adirkuhn/commerceos/api/reviews"
	genscannerauth "github.com/adirkuhn/commerceos/api/scannerauth"
	genscanning "github.com/adirkuhn/commerceos/api/scanning"
	genscheduler "github.com/adirkuhn/commerceos/api/scheduler"
	gensearch "github.com/adirkuhn/commerceos/api/search"
	gensettings "github.com/adirkuhn/commerceos/api/settings"
	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	genstockexpiry "github.com/adirkuhn/commerceos/api/stockexpiry"
	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	gensupplierissues "github.com/adirkuhn/commerceos/api/supplierissues"
	gensuppliers "github.com/adirkuhn/commerceos/api/suppliers"
	gentax "github.com/adirkuhn/commerceos/api/tax"
	gentheme "github.com/adirkuhn/commerceos/api/theme"
	genweightportions "github.com/adirkuhn/commerceos/api/weightportions"
	genwhatsapp "github.com/adirkuhn/commerceos/api/whatsapp"
	"github.com/adirkuhn/commerceos/domain/messaging"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	bolconfirmationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/bolconfirmations"
	collectionshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/collections"
	couponshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/coupons"
	customershandler "github.com/adirkuhn/commerceos/infrastructure/handlers/customers"
	dashboardhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/dashboard"
	deadlettershandler "github.com/adirkuhn/commerceos/infrastructure/handlers/deadletters"
	emailintegrationhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/emailintegration"
	fxhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/fx"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/iam"
	"github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
	inventoryhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/inventory"
	messagecenterhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/messagecenter"
	moneybirdhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/moneybird"
	newsletterhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/newsletter"
	notificationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/notifications"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
	orderpaymentshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/orderpayments"
	pmshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pms"
	poshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/pos"
	procurementhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/procurement"
	promotionshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/promotions"
	purchaseplanninghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/purchaseplanning"
	quoteshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/quotes"
	reportshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/reports"
	reviewshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/reviews"
	scannerauthhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/scannerauth"
	scanninghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/scanning"
	schedulerhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/scheduler"
	searchhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/search"
	settingshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/settings"
	shippinghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/shipping"
	stockexpiryhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/stockexpiry"
	storefronthandler "github.com/adirkuhn/commerceos/infrastructure/handlers/storefront"
	supplierissueshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/supplierissues"
	supplierhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/suppliers"
	taxhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/tax"
	themehandler "github.com/adirkuhn/commerceos/infrastructure/handlers/theme"
	webhookshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/webhooks"
	weightportionshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/weightportions"
	whatsapphandler "github.com/adirkuhn/commerceos/infrastructure/handlers/whatsapp"
	"github.com/adirkuhn/commerceos/infrastructure/lm2license"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/noop"
	"github.com/adirkuhn/commerceos/infrastructure/messaging/smtp"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/storage"
	"github.com/adirkuhn/commerceos/infrastructure/woocommerce"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
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

	db, err := db.Open(cfg.DBDsn, 20)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	// The webhook-ingestion hot path (see infrastructure/handlers/webhooks)
	// publishes to JetStream synchronously in-request -- unlike every other
	// domain event in this app, which goes through the transactional
	// outbox and cmd/relay. RetryOnFailedConnect+MaxReconnects(-1): if
	// NATS/JetStream is down at startup or drops mid-run, Connect returns
	// immediately and the client keeps retrying in the background, so the
	// API stays up (individual webhook publishes fail over to the DB
	// fallback table -- see application/webhooks.ReceiveWebhookUseCase --
	// rather than the whole process crash-looping).
	nc, err := nats.Connect(cfg.NatsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Drain()
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}
	if err := publishers.EnsureWebhookStream(js); err != nil {
		log.Fatalf("jetstream: %v", err)
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
		cfg.IntegrationsEncryptionSecret, cfg.PublicAPIBaseURL, cfg.FrontendBaseURL, publishers.NewJetStreamPublisher(nc, js))

	// Own unauthenticated route, mounted on the root router like the
	// WooCommerce connect callback above -- see infrastructure/handlers/
	// webhooks's doc comment for why.
	webhookshandler.Register(r, db, repositories.NewConnectionRepo(db),
		publishers.NewJetStreamPublisher(nc, js), repositories.NewWebhookFallbackRepo(db), cfg.IntegrationsEncryptionSecret)

	shippingSpec, err := genshipping.GetSpec()
	if err != nil {
		log.Fatalf("shipping: load openapi spec: %v", err)
	}
	// Own sub-group so the shipping-only spec validator never applies to
	// iam/integrations/oms routes -- same reasoning as pmsGroup below.
	shippingGroup := v1.Group("")
	shippingGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(shippingSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	shippinghandler.Register(shippingGroup, db, cfg.IntegrationsEncryptionSecret, cfg.PublicAPIBaseURL)

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
	inventoryhandler.Register(inventoryGroup, db, outboxRepo, fileStorage)

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
	// Tenant-wide settings: every role may read, only owners/admins change
	// them (customer_user is read-only -- see CLAUDE.md Auth & RBAC).
	settingsGroup.Use(middleware.RequireRoleForWrites(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	settingshandler.Register(settingsGroup, db, outboxRepo)

	collectionsSpec, err := gencollections.GetSpec()
	if err != nil {
		log.Fatalf("collections: load openapi spec: %v", err)
	}
	collectionsGroup := v1.Group("")
	collectionsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(collectionsSpec, &ginmiddleware.Options{SilenceServersWarning: true}))
	collectionshandler.Register(collectionsGroup, collectionshandler.NewHandler(collectionshandler.NewUseCases(db)))

	taxSpec, err := gentax.GetSpec()
	if err != nil {
		log.Fatalf("tax: load openapi spec: %v", err)
	}
	// Own sub-group so the tax-only spec validator never applies to the
	// other modules' routes — same reasoning as pmsGroup above.
	taxGroup := v1.Group("")
	taxGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(taxSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	taxhandler.Register(taxGroup, db)

	promotionsSpec, err := genpromotions.GetSpec()
	if err != nil {
		log.Fatalf("promotions: load openapi spec: %v", err)
	}
	// Own sub-group so the promotions-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	promotionsGroup := v1.Group("")
	promotionsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(promotionsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	promotionshandler.Register(promotionsGroup, db)

	couponsSpec, err := gencoupons.GetSpec()
	if err != nil {
		log.Fatalf("coupons: load openapi spec: %v", err)
	}
	// Own sub-group so the coupons-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	couponsGroup := v1.Group("")
	couponsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(couponsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	couponshandler.Register(couponsGroup, db)

	fxSpec, err := genfx.GetSpec()
	if err != nil {
		log.Fatalf("fx: load openapi spec: %v", err)
	}
	// Own sub-group so the fx-only spec validator never applies to the
	// other modules' routes — same reasoning as pmsGroup above.
	fxGroup := v1.Group("")
	fxGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(fxSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	fxhandler.Register(fxGroup, db)

	reviewsSpec, err := genreviews.GetSpec()
	if err != nil {
		log.Fatalf("reviews: load openapi spec: %v", err)
	}
	// Own sub-group so the reviews-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	reviewsGroup := v1.Group("")
	reviewsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(reviewsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	reviewshandler.Register(reviewsGroup, db)

	whatsappSpec, err := genwhatsapp.GetSpec()
	if err != nil {
		log.Fatalf("whatsapp: load openapi spec: %v", err)
	}
	// Own sub-group so the whatsapp-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	whatsappGroup := v1.Group("")
	whatsappGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(whatsappSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	whatsapphandler.Register(whatsappGroup, db)

	messagecenterSpec, err := genmessagecenter.GetSpec()
	if err != nil {
		log.Fatalf("messagecenter: load openapi spec: %v", err)
	}
	// Own sub-group so the messagecenter-only spec validator never applies
	// to the other modules' routes — same reasoning as pmsGroup above.
	messagecenterGroup := v1.Group("")
	messagecenterGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(messagecenterSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	messagecenterhandler.Register(messagecenterGroup, db, fileStorage)

	emailintegrationSpec, err := genemailintegration.GetSpec()
	if err != nil {
		log.Fatalf("emailintegration: load openapi spec: %v", err)
	}
	// Own sub-group so the emailintegration-only spec validator never
	// applies to the other modules' routes — same reasoning as pmsGroup
	// above. Kept as its own small module rather than folded into
	// api/integrations' already large composed handler -- see
	// api/emailintegration/openapi.yaml's doc comment.
	emailintegrationGroup := v1.Group("")
	emailintegrationGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(emailintegrationSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	emailintegrationhandler.Register(emailintegrationGroup, db, outboxRepo, cfg.IntegrationsEncryptionSecret)

	weightportionsSpec, err := genweightportions.GetSpec()
	if err != nil {
		log.Fatalf("weightportions: load openapi spec: %v", err)
	}
	// Own sub-group so the weightportions-only spec validator never applies
	// to the other modules' routes — same reasoning as pmsGroup above.
	weightportionsGroup := v1.Group("")
	weightportionsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(weightportionsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	weightportionshandler.Register(weightportionsGroup, db, outboxRepo)

	moneybirdSpec, err := genmoneybird.GetSpec()
	if err != nil {
		log.Fatalf("moneybird: load openapi spec: %v", err)
	}
	// Own sub-group so the moneybird-only spec validator never applies to
	// the other modules' routes. Role guard: see moneybirdhandler.RegisterRoutes.
	moneybirdGroup := v1.Group("")
	moneybirdGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(moneybirdSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	moneybirdhandler.Register(moneybirdGroup, db, outboxRepo, cfg.IntegrationsEncryptionSecret)

	stockexpirySpec, err := genstockexpiry.GetSpec()
	if err != nil {
		log.Fatalf("stockexpiry: load openapi spec: %v", err)
	}
	// Own sub-group so the stockexpiry-only spec validator never applies
	// to the other modules' routes — same reasoning as pmsGroup above.
	// No RequireRole, like inventory: any authenticated role can work the
	// close-to-expiry list.
	stockexpiryGroup := v1.Group("")
	stockexpiryGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(stockexpirySpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	stockexpiryhandler.Register(stockexpiryGroup, db, outboxRepo)

	bolconfirmationsSpec, err := genbolconfirmations.GetSpec()
	if err != nil {
		log.Fatalf("bolconfirmations: load openapi spec: %v", err)
	}
	// Own sub-group so the bolconfirmations-only spec validator never
	// applies to the other modules' routes — same reasoning as pmsGroup
	// above. Read-only, no RequireRole: any authenticated role may see an
	// order's Bol confirmation status; tenant RLS is the isolation boundary.
	bolconfirmationsGroup := v1.Group("")
	bolconfirmationsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(bolconfirmationsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	bolconfirmationshandler.Register(bolconfirmationsGroup, db)

	orderpaymentsSpec, err := genorderpayments.GetSpec()
	if err != nil {
		log.Fatalf("orderpayments: load openapi spec: %v", err)
	}
	// Own sub-group so the orderpayments-only spec validator never applies
	// to the other modules' routes — same reasoning as pmsGroup above.
	orderpaymentsGroup := v1.Group("")
	// It rewrites orders' payment info -- not for read-only customer_user.
	orderpaymentsGroup.Use(middleware.RequireRole(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	orderpaymentsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(orderpaymentsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	orderpaymentshandler.Register(orderpaymentsGroup, db, wooFactory, cfg.IntegrationsEncryptionSecret)

	reportsSpec, err := genreports.GetSpec()
	if err != nil {
		log.Fatalf("reports: load openapi spec: %v", err)
	}
	// Own sub-group so the reports-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	reportsGroup := v1.Group("")
	reportsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(reportsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	reportshandler.Register(reportsGroup, db)

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

	schedulerSpec, err := genscheduler.GetSpec()
	if err != nil {
		log.Fatalf("scheduler: load openapi spec: %v", err)
	}
	// Own sub-group, gated to super_admin: every route here is a
	// cross-tenant platform-operational view/action (see
	// api/scheduler/openapi.yaml's doc), not tenant self-service settings.
	schedulerGroup := v1.Group("")
	schedulerGroup.Use(middleware.RequireRole(middleware.RoleSuperAdmin))
	schedulerGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(schedulerSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	schedulerhandler.Register(schedulerGroup, db)

	// Dead-letter admin (COM-14): super_admin only, spec-validated -- both
	// applied inside Register. Reads the DLQ streams by sequence, never
	// through a consumer, so the API process consumes nothing.
	jsAPI, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("deadletters: jetstream: %v", err)
	}
	if err := deadlettershandler.Register(v1, db, jsAPI, outboxRepo); err != nil {
		log.Fatalf("deadletters: %v", err)
	}

	suppliersSpec, err := gensuppliers.GetSpec()
	if err != nil {
		log.Fatalf("suppliers: load openapi spec: %v", err)
	}
	// Own sub-group so the suppliers-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	suppliersGroup := v1.Group("")
	suppliersGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(suppliersSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	supplierhandler.Register(suppliersGroup, db)

	customersSpec, err := gencustomers.GetSpec()
	if err != nil {
		log.Fatalf("customers: load openapi spec: %v", err)
	}
	// Own sub-group so the customers-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	customersGroup := v1.Group("")
	customersGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(customersSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	customershandler.Register(customersGroup, db)

	dashboardSpec, err := gendashboard.GetSpec()
	if err != nil {
		log.Fatalf("dashboard: load openapi spec: %v", err)
	}
	// Own sub-group so the dashboard-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	// Read-only, no RequireRole -- any authenticated role can view their
	// own tenant's summary figures.
	dashboardGroup := v1.Group("")
	dashboardGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(dashboardSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	dashboardhandler.Register(dashboardGroup, db)

	searchSpec, err := gensearch.GetSpec()
	if err != nil {
		log.Fatalf("search: load openapi spec: %v", err)
	}
	// Own sub-group so the search-only spec validator never applies to the
	// other modules' routes — same reasoning as pmsGroup above. Read-only,
	// no RequireRole -- same "tenant RLS is the isolation boundary" call as
	// dashboardGroup above; global search has no fine-grained per-entity
	// RBAC in this codebase to gate on yet (see CLAUDE.md's documented but
	// unimplemented user_permissions convention).
	searchGroup := v1.Group("")
	searchGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(searchSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	searchhandler.Register(searchGroup, db)

	omsSpec, err := genoms.GetSpec()
	if err != nil {
		log.Fatalf("oms: load openapi spec: %v", err)
	}
	// Own sub-group so the oms-only spec validator never applies to the
	// other modules' routes — same reasoning as pmsGroup above. Read-only
	// (list/get sales orders) -- the write side is api/pos, not here, see
	// api/oms/generate.go's doc comment.
	omsGroup := v1.Group("")
	omsGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(omsSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	omshandler.Register(omsGroup, db, cfg.IntegrationsEncryptionSecret)

	procurementSpec, err := genprocurement.GetSpec()
	if err != nil {
		log.Fatalf("procurement: load openapi spec: %v", err)
	}
	// Own sub-group so the procurement-only spec validator never applies to
	// the other modules' routes — same reasoning as pmsGroup above.
	procurementGroup := v1.Group("")
	procurementGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(procurementSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	procurementhandler.Register(procurementGroup, db, fileStorage)

	purchaseplanningSpec, err := genpurchaseplanning.GetSpec()
	if err != nil {
		log.Fatalf("purchaseplanning: load openapi spec: %v", err)
	}
	// Own sub-group so the purchaseplanning-only spec validator never
	// applies to the other modules' routes — same reasoning as pmsGroup above.
	purchaseplanningGroup := v1.Group("")
	// Every role may see the plan; only owners/admins turn it into POs.
	purchaseplanningGroup.Use(middleware.RequireRoleForWrites(middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleSuperAdmin))
	purchaseplanningGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(purchaseplanningSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	purchaseplanninghandler.Register(purchaseplanningGroup, db, outboxRepo)

	scanningSpec, err := genscanning.GetSpec()
	if err != nil {
		log.Fatalf("scanning: load openapi spec: %v", err)
	}
	// Authenticated (JWT via v1) but deliberately NOT admin-restricted --
	// see api/scanning/generate.go's doc comment: any authenticated role
	// should be able to pair their own scanner.
	scanningGroup := v1.Group("")
	scanningGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(scanningSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	scanninghandler.Register(scanningGroup, db, []byte(cfg.JWTSecret))

	// Scanner claim is the one genuinely public, unauthenticated route in
	// this API -- mounted on its own /api/v1 group with no JWT middleware,
	// same reasoning as the /api/v1/uploads static mount below (both need
	// to be reachable before any token exists). See
	// api/scannerauth/generate.go's doc comment.
	scannerauthSpec, err := genscannerauth.GetSpec()
	if err != nil {
		log.Fatalf("scannerauth: load openapi spec: %v", err)
	}
	scannerauthGroup := r.Group("/api/v1")
	scannerauthGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(scannerauthSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	scannerauthhandler.Register(scannerauthGroup, db, []byte(cfg.JWTSecret))

	// Storefront API (cart recommendations first) -- authenticated by a
	// per-connection storefront key, not the human JWT (the shopper's
	// browser never has a CommerceOS session), so it's its own /api/v1
	// group with no JWT middleware, same reasoning as scannerauthGroup
	// above. See api/storefront/generate.go's doc comment.
	storefrontSpec, err := genstorefront.GetSpec()
	if err != nil {
		log.Fatalf("storefront: load openapi spec: %v", err)
	}
	storefrontGroup := r.Group("/api/v1")
	storefrontGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(storefrontSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	var emailClient messaging.Client = noop.Client{}
	if cfg.SMTPHost != "" {
		emailClient = smtp.NewClient(smtp.Config{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		})
	}
	storefronthandler.Register(storefrontGroup, db, cfg.IntegrationsEncryptionSecret, cfg.FrontendBaseURL, cfg.StorefrontPublicURL, emailClient)

	// Storefront themes (api/theme): the admin library sits behind the JWT
	// like every other admin route; the storefront-hosted editor and the
	// public storefront reads authenticate their own bearer credential
	// (edit session / channel storefront key), so they get a JWT-free group.
	newsletterSpec, err := gennewsletter.GetSpec()
	if err != nil {
		log.Fatalf("newsletter: load openapi spec: %v", err)
	}
	newsletterGroup := v1.Group("")
	newsletterGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(newsletterSpec, &ginmiddleware.Options{SilenceServersWarning: true}))
	newsletterhandler.Register(newsletterGroup, newsletterhandler.NewHandler(newsletterhandler.NewUseCases(
		newsletterhandler.NewDeps(db, cfg.IntegrationsEncryptionSecret, emailClient, cfg.StorefrontPublicURL))))

	supplierissuesSpec, err := gensupplierissues.GetSpec()
	if err != nil {
		log.Fatalf("supplierissues: load openapi spec: %v", err)
	}
	// Own sub-group so the supplierissues-only spec validator never applies
	// to the other modules' routes — same reasoning as pmsGroup above. No
	// RequireRole, like procurement: any authenticated role works purchase
	// orders.
	supplierissuesGroup := v1.Group("")
	supplierissuesGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(supplierissuesSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	supplierissueshandler.Register(supplierissuesGroup, db, outboxRepo, cfg.IntegrationsEncryptionSecret, emailClient)

	quotesSpec, err := genquotes.GetSpec()
	if err != nil {
		log.Fatalf("quotes: load openapi spec: %v", err)
	}
	// Own sub-group so the quotes-only spec validator never applies to the
	// other modules' routes. No RequireRole: every authenticated role may
	// work quotes for now (owner's call, 2026-10-23).
	quotesGroup := v1.Group("")
	quotesGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(quotesSpec, &ginmiddleware.Options{
		SilenceServersWarning: true,
	}))
	quoteshandler.Register(quotesGroup, db, outboxRepo)

	themeSpec, err := gentheme.GetSpec()
	if err != nil {
		log.Fatalf("theme: load openapi spec: %v", err)
	}
	themeUseCases, err := themehandler.NewUseCases(db, fileStorage)
	if err != nil {
		log.Fatalf("theme: %v", err)
	}
	themeHandler := themehandler.NewHandler(themeUseCases)
	themeValidator := ginmiddleware.OapiRequestValidatorWithOptions(themeSpec, &ginmiddleware.Options{SilenceServersWarning: true})
	themeAdminGroup := v1.Group("")
	themeAdminGroup.Use(themeValidator)
	themehandler.RegisterAdmin(themeAdminGroup, themeHandler)
	themePublicGroup := r.Group("/api/v1")
	themePublicGroup.Use(themeValidator)
	themehandler.RegisterPublic(themePublicGroup, themeHandler)

	// POS desktop integration -- authenticated by the caller's lm2 license
	// directly (middleware.POSLicense), never the human JWT, so it's its
	// own /api/v1 group with no JWT middleware, same reasoning as
	// scannerauthGroup above. Only mounted when LM2_LICENSE_PUBLIC_KEY_URL
	// is configured -- see Config.LM2LicensePublicKeyURL's doc comment.
	if cfg.LM2LicensePublicKeyURL == "" {
		log.Printf("pos: LM2_LICENSE_PUBLIC_KEY_URL not set, /pos/* routes not mounted")
	} else {
		posSpec, err := genpos.GetSpec()
		if err != nil {
			log.Fatalf("pos: load openapi spec: %v", err)
		}
		licenseVerifier := lm2license.NewTokenService(lm2license.NewHTTPPublicKeyFetcher(cfg.LM2LicensePublicKeyURL))
		posGroup := r.Group("/api/v1")
		// Logs every /pos/* request's method/path/body before anything
		// else runs, so a request that fails license auth or OpenAPI
		// validation still shows up in the log -- useful for debugging a
		// POS client sending a bad request, not just successful ones.
		posGroup.Use(middleware.RequestBodyLogger())
		posGroup.Use(middleware.POSLicense(licenseVerifier))
		posGroup.Use(ginmiddleware.OapiRequestValidatorWithOptions(posSpec, &ginmiddleware.Options{
			SilenceServersWarning: true,
		}))
		poshandler.Register(posGroup, db)
	}

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
