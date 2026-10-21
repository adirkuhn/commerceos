// Package shipping is the Gin adapter for the MyParcel shipping-label
// module -- thin per CLAUDE.md: parse request, call the use case, map the
// result/error to a response. No business logic here.
package shipping

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/carriers"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
	"github.com/adirkuhn/commerceos/infrastructure/pdok"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// The interfaces below are each satisfied by the real use case
// (application/shipping / application/integrations), narrowed to just
// Execute so the handler can be unit tested with stubs.
type connectMyParcelUseCase interface {
	Execute(ctx context.Context, in appintegrations.ConnectMyParcelInput) (*integrationsDomain.Connection, error)
}
type registerWebhookUseCase interface {
	Execute(ctx context.Context, connectionID string) error
}
type getSettingsUseCase interface {
	Execute(ctx context.Context, connectionID string) (*appshipping.ShippingSettingsResult, error)
}
type updateSettingsUseCase interface {
	Execute(ctx context.Context, in appshipping.UpdateShippingSettingsInput) (*shippingDomain.ShippingConnectionSettings, error)
}
type upsertOverrideUseCase interface {
	Execute(ctx context.Context, in appshipping.UpsertCarrierOverrideInput) (*shippingDomain.ShippingCarrierOverride, error)
}
type createShipmentUseCase interface {
	Execute(ctx context.Context, in appshipping.CreateShipmentInput) (*shippingDomain.Shipment, error)
}
type getLabelUseCase interface {
	Execute(ctx context.Context, shipmentID, size string) (*integrationsDomain.LabelResult, error)
}
type listShipmentsUseCase interface {
	Execute(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error)
}
type exportMyParcelOrderUseCase interface {
	Execute(ctx context.Context, in appshipping.ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error)
}
type listTrackingEventsUseCase interface {
	Execute(ctx context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error)
}
type refreshTrackingUseCase interface {
	Execute(ctx context.Context, shipmentID string) (shippingDomain.ShipmentStatusEvent, error)
}
type lookupAddressUseCase interface {
	Execute(ctx context.Context, postalCode, houseNumber string) (*shippingDomain.AddressLookupResult, error)
}
type getDeliveryPriceEstimateUseCase interface {
	Execute(ctx context.Context, in appshipping.GetDeliveryPriceEstimateInput) ([]shippingDomain.DeliveryPriceOption, error)
}
type cancelShipmentUseCase interface {
	Execute(ctx context.Context, in appshipping.CancelShipmentInput) (*appshipping.CancelShipmentResult, error)
}

// Handler implements genshipping.ServerInterface.
type Handler struct {
	connect             connectMyParcelUseCase
	registerWebhook     registerWebhookUseCase
	getSettings         getSettingsUseCase
	updateSettings      updateSettingsUseCase
	upsertOverride      upsertOverrideUseCase
	createShipment      createShipmentUseCase
	getLabel            getLabelUseCase
	listShipments       listShipmentsUseCase
	exportMyParcelOrder exportMyParcelOrderUseCase
	listTrackingEvents  listTrackingEventsUseCase
	refreshTracking     refreshTrackingUseCase
	lookupAddress       lookupAddressUseCase
	cancelShipment      cancelShipmentUseCase
	getDeliveryEstimate getDeliveryPriceEstimateUseCase
	*PreferencesHandler
}

func NewHandler(
	connect connectMyParcelUseCase,
	registerWebhook registerWebhookUseCase,
	getSettings getSettingsUseCase,
	updateSettings updateSettingsUseCase,
	upsertOverride upsertOverrideUseCase,
	createShipment createShipmentUseCase,
	getLabel getLabelUseCase,
	listShipments listShipmentsUseCase,
	exportMyParcelOrder exportMyParcelOrderUseCase,
	listTrackingEvents listTrackingEventsUseCase,
	refreshTracking refreshTrackingUseCase,
	lookupAddress lookupAddressUseCase,
	cancelShipment cancelShipmentUseCase,
	getDeliveryEstimate getDeliveryPriceEstimateUseCase,
	preferences *PreferencesHandler,
) *Handler {
	return &Handler{
		connect: connect, registerWebhook: registerWebhook, getSettings: getSettings, updateSettings: updateSettings,
		upsertOverride: upsertOverride, createShipment: createShipment, getLabel: getLabel, listShipments: listShipments,
		exportMyParcelOrder: exportMyParcelOrder, listTrackingEvents: listTrackingEvents, refreshTracking: refreshTracking,
		lookupAddress: lookupAddress, cancelShipment: cancelShipment, getDeliveryEstimate: getDeliveryEstimate,
		PreferencesHandler: preferences,
	}
}

// Register is the production wiring.
func Register(rg gin.IRouter, db *gorm.DB, encryptionSecret, publicAPIBaseURL string) {
	connRepo := repositories.NewConnectionRepo(db)
	settingsRepo := repositories.NewShippingConnectionSettingsRepo(db)
	tenantSettingsRepo := repositories.NewTenantSettingsRepo(db)
	overrideRepo := repositories.NewShippingCarrierOverrideRepo(db)
	shipmentRepo := repositories.NewShipmentRepo(db)
	statusEventRepo := repositories.NewShipmentStatusEventRepo(db)
	fulfilmentOrderRepo := repositories.NewMyParcelFulfilmentOrderRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	preferencesRepo := repositories.NewShippingPreferencesRepo(db)
	myParcelFactory := myparcel.NewClientFactory()
	carrierFactory := carriers.NewDefaultRegistry()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	recordActivity := appoms.NewRecordSalesOrderActivityUseCase(db, salesOrderRepo, repositories.NewSalesOrderActivityRepo(db), outboxRepo)
	exportMyParcelOrderUC := appshipping.NewExportMyParcelOrderUseCase(db, connRepo, settingsRepo, fulfilmentOrderRepo, salesOrderRepo, lineItemRepo, myParcelFactory, encryptionSecret, recordActivity, outboxRepo)

	h := NewHandler(
		appintegrations.NewConnectMyParcelUseCase(db, connRepo, settingsRepo, outboxRepo, encryptionSecret),
		appshipping.NewRegisterMyParcelWebhookUseCase(db, connRepo, myParcelFactory, encryptionSecret, publicAPIBaseURL),
		appshipping.NewGetShippingSettingsUseCase(db, settingsRepo, overrideRepo),
		appshipping.NewUpdateShippingSettingsUseCase(db, settingsRepo, outboxRepo),
		appshipping.NewUpsertCarrierOverrideUseCase(db, overrideRepo, outboxRepo),
		appshipping.NewCreateShipmentUseCase(db, connRepo, settingsRepo, overrideRepo, shipmentRepo, carrierFactory, encryptionSecret, recordActivity, tenantSettingsRepo, salesOrderRepo, exportMyParcelOrderUC, lineItemRepo, skuRepo, outboxRepo),
		appshipping.NewGetShipmentLabelUseCase(db, shipmentRepo, connRepo, settingsRepo, carrierFactory, encryptionSecret),
		appshipping.NewListShipmentsForOrderUseCase(db, shipmentRepo, overrideRepo),
		exportMyParcelOrderUC,
		appshipping.NewListShipmentTrackingEventsUseCase(db, statusEventRepo),
		appshipping.NewRefreshShipmentTrackingUseCase(db, shipmentRepo, statusEventRepo, connRepo, salesOrderRepo, carrierFactory, encryptionSecret),
		appshipping.NewLookupAddressUseCase(pdok.NewClient(pdok.Config{})),
		appshipping.NewCancelShipmentUseCase(db, connRepo, shipmentRepo, carrierFactory, encryptionSecret, recordActivity, outboxRepo),
		appshipping.NewGetDeliveryPriceEstimateUseCase(db, connRepo, carrierFactory, encryptionSecret),
		NewPreferencesHandler(
			appshipping.NewListShippingConnectionsUseCase(db, connRepo, preferencesRepo, carrierFactory),
			appshipping.NewSetDefaultShippingConnectionUseCase(db, connRepo, preferencesRepo, carrierFactory, outboxRepo),
		),
	)
	genshipping.RegisterHandlersWithOptions(rg, h, genshipping.GinServerOptions{})
}
