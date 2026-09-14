// Package shipping is the Gin adapter for the MyParcel shipping-label
// module -- thin per CLAUDE.md: parse request, call the use case, map the
// result/error to a response. No business logic here.
package shipping

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
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
) *Handler {
	return &Handler{
		connect: connect, registerWebhook: registerWebhook, getSettings: getSettings, updateSettings: updateSettings,
		upsertOverride: upsertOverride, createShipment: createShipment, getLabel: getLabel, listShipments: listShipments,
		exportMyParcelOrder: exportMyParcelOrder, listTrackingEvents: listTrackingEvents, refreshTracking: refreshTracking,
		lookupAddress: lookupAddress, cancelShipment: cancelShipment, getDeliveryEstimate: getDeliveryEstimate,
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message}})
}

func mapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, shippingDomain.ErrShipmentNotFound), errors.Is(err, integrationsDomain.ErrNoTenantContext),
		errors.Is(err, shippingDomain.ErrAddressNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, shippingDomain.ErrMissingRecipient), errors.Is(err, shippingDomain.ErrInvalidAddressLookupInput):
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, shippingDomain.ErrWebhookSecretNotSet):
		writeError(c, http.StatusConflict, "webhook_secret_not_set", err.Error())
	case errors.Is(err, shippingDomain.ErrDeliveryEstimateUnsupportedCountry):
		writeError(c, http.StatusUnprocessableEntity, "unsupported_country", err.Error())
	default:
		writeError(c, http.StatusBadGateway, "upstream_error", err.Error())
	}
}

func (h *Handler) ConnectMyParcel(c *gin.Context) {
	var req genshipping.ConnectMyParcelJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	conn, err := h.connect.Execute(c.Request.Context(), appintegrations.ConnectMyParcelInput{APIKey: req.ApiKey})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"connection_id": conn.ID, "provider": string(conn.Provider), "status": string(conn.Status),
	}, "error": nil})
}

func (h *Handler) RegisterMyParcelWebhook(c *gin.Context, connectionId genshipping.ConnectionId) {
	if err := h.registerWebhook.Execute(c.Request.Context(), connectionId.String()); err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"ok": true}, "error": nil})
}

func (h *Handler) GetShippingSettings(c *gin.Context, connectionId genshipping.ConnectionId) {
	result, err := h.getSettings.Execute(c.Request.Context(), connectionId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"settings": settingsToResponse(result.Settings), "carrier_overrides": overridesToResponse(result.CarrierOverrides),
	}, "error": nil})
}

func (h *Handler) UpdateShippingSettings(c *gin.Context, connectionId genshipping.ConnectionId) {
	var req genshipping.UpdateShippingSettingsJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	onlyRecipient, sig := false, false
	if req.DefaultOnlyRecipient != nil {
		onlyRecipient = *req.DefaultOnlyRecipient
	}
	if req.DefaultSignatureRequired != nil {
		sig = *req.DefaultSignatureRequired
	}
	autoPrint := false
	if req.AutoPrintEnabled != nil {
		autoPrint = *req.AutoPrintEnabled
	}
	insureByDefault := false
	if req.InsureByDefault != nil {
		insureByDefault = *req.InsureByDefault
	}
	labelSize := ""
	if req.DefaultLabelSize != nil {
		labelSize = string(*req.DefaultLabelSize)
	}
	exportOrder := false
	if req.ExportOrderToMyparcel != nil {
		exportOrder = *req.ExportOrderToMyparcel
	}
	settings, err := h.updateSettings.Execute(c.Request.Context(), appshipping.UpdateShippingSettingsInput{
		ConnectionID: connectionId.String(), DefaultCarrier: shippingDomain.Carrier(req.DefaultCarrier),
		DefaultPackageType: shippingDomain.PackageType(req.DefaultPackageType), DefaultLabelFormat: string(req.DefaultLabelFormat),
		DefaultLabelSize:         labelSize,
		DefaultSignatureRequired: sig, DefaultOnlyRecipient: onlyRecipient,
		InsureByDefault:   insureByDefault,
		MyParcelAccountID: req.MyparcelAccountId, MyParcelShopID: req.MyparcelShopId, AutoPrintEnabled: autoPrint,
		ExportOrderToMyParcel: exportOrder,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settingsToResponse(settings), "error": nil})
}

func (h *Handler) UpsertCarrierOverride(c *gin.Context, connectionId genshipping.ConnectionId, carrier int) {
	var req genshipping.UpsertCarrierOverrideJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	var packageType *shippingDomain.PackageType
	if req.PackageType != nil {
		pt := shippingDomain.PackageType(*req.PackageType)
		packageType = &pt
	}
	override, err := h.upsertOverride.Execute(c.Request.Context(), appshipping.UpsertCarrierOverrideInput{
		ConnectionID: connectionId.String(), Carrier: shippingDomain.Carrier(carrier),
		PackageType: packageType, SignatureRequired: req.SignatureRequired, OnlyRecipient: req.OnlyRecipient,
		InsureByDefault: req.InsureByDefault, TrackingURLTemplate: req.TrackingUrlTemplate,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": overrideToResponse(*override), "error": nil})
}

func (h *Handler) CreateShipment(c *gin.Context) {
	var req genshipping.CreateShipmentJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	in := appshipping.CreateShipmentInput{
		SalesOrderID: req.SalesOrderId.String(), ConnectionID: req.ConnectionId.String(), Carrier: shippingDomain.Carrier(req.Carrier),
		Recipient: appshipping.RecipientInput{
			Person: req.Recipient.Person, Street: req.Recipient.Street, City: req.Recipient.City, CC: req.Recipient.Cc,
		},
		UserID:            userID,
		SignatureRequired: req.SignatureRequired, OnlyRecipient: req.OnlyRecipient,
	}
	if req.Recipient.Company != nil {
		in.Recipient.Company = *req.Recipient.Company
	}
	if req.Recipient.Number != nil {
		in.Recipient.Number = *req.Recipient.Number
	}
	if req.Recipient.NumberSuffix != nil {
		in.Recipient.NumberSuffix = *req.Recipient.NumberSuffix
	}
	if req.Recipient.PostalCode != nil {
		in.Recipient.PostalCode = *req.Recipient.PostalCode
	}
	if req.Recipient.Email != nil {
		in.Recipient.Email = *req.Recipient.Email
	}
	if req.Recipient.Phone != nil {
		in.Recipient.Phone = *req.Recipient.Phone
	}
	if req.PackageType != nil {
		pt := shippingDomain.PackageType(*req.PackageType)
		in.PackageType = &pt
	}
	if req.LabelFormat != nil {
		in.LabelFormat = string(*req.LabelFormat)
	}

	shipment, err := h.createShipment.Execute(c.Request.Context(), in)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": shipmentToResponse(*shipment), "error": nil})
}

func (h *Handler) ExportMyParcelOrder(c *gin.Context, connectionId openapi_types.UUID, salesOrderId openapi_types.UUID) {
	var req genshipping.ExportMyParcelOrderJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	in := appshipping.ExportMyParcelOrderInput{
		SalesOrderID: salesOrderId.String(), ConnectionID: connectionId.String(), Carrier: shippingDomain.Carrier(req.Carrier),
		Recipient: appshipping.RecipientInput{
			Person: req.Recipient.Person, Street: req.Recipient.Street, City: req.Recipient.City, CC: req.Recipient.Cc,
		},
		UserID: userID,
	}
	if req.Recipient.Company != nil {
		in.Recipient.Company = *req.Recipient.Company
	}
	if req.Recipient.Number != nil {
		in.Recipient.Number = *req.Recipient.Number
	}
	if req.Recipient.NumberSuffix != nil {
		in.Recipient.NumberSuffix = *req.Recipient.NumberSuffix
	}
	if req.Recipient.PostalCode != nil {
		in.Recipient.PostalCode = *req.Recipient.PostalCode
	}
	if req.Recipient.Email != nil {
		in.Recipient.Email = *req.Recipient.Email
	}
	if req.Recipient.Phone != nil {
		in.Recipient.Phone = *req.Recipient.Phone
	}

	order, err := h.exportMyParcelOrder.Execute(c.Request.Context(), in)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": genshipping.MyParcelFulfilmentOrder{
		Id: &order.ID, ConnectionId: &order.ConnectionID, SalesOrderId: &order.SalesOrderID,
		MyparcelUuid: &order.MyParcelUUID, Status: &order.Status,
	}, "error": nil})
}

func (h *Handler) GetShipmentLabel(c *gin.Context, shipmentId openapi_types.UUID, params genshipping.GetShipmentLabelParams) {
	size := ""
	if params.Size != nil {
		size = string(*params.Size)
	}
	label, err := h.getLabel.Execute(c.Request.Context(), shipmentId.String(), size)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.Data(http.StatusOK, label.ContentType, label.Data)
}

func (h *Handler) ListShipmentsForOrder(c *gin.Context, salesOrderId openapi_types.UUID) {
	shipments, err := h.listShipments.Execute(c.Request.Context(), salesOrderId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	out := make([]genshipping.Shipment, len(shipments))
	for i, s := range shipments {
		out[i] = shipmentToResponse(s)
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "error": nil})
}

func (h *Handler) ListShipmentTrackingEvents(c *gin.Context, shipmentId openapi_types.UUID) {
	events, err := h.listTrackingEvents.Execute(c.Request.Context(), shipmentId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	out := make([]genshipping.ShipmentTrackingEvent, len(events))
	for i, e := range events {
		out[i] = trackingEventToResponse(e)
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "error": nil})
}

func (h *Handler) RefreshShipmentTracking(c *gin.Context, shipmentId openapi_types.UUID) {
	event, err := h.refreshTracking.Execute(c.Request.Context(), shipmentId.String())
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": trackingEventToResponse(event), "error": nil})
}

func (h *Handler) CancelShipment(c *gin.Context, shipmentId openapi_types.UUID) {
	userID := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		userID = claims.UserID
	}
	result, err := h.cancelShipment.Execute(c.Request.Context(), appshipping.CancelShipmentInput{
		ShipmentID: shipmentId.String(), UserID: userID,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": genshipping.CancelShipmentResult{
		DeletedOnMyparcel: &result.DeletedOnMyParcel, Message: &result.Message,
	}, "error": nil})
}

func (h *Handler) LookupAddress(c *gin.Context, params genshipping.LookupAddressParams) {
	result, err := h.lookupAddress.Execute(c.Request.Context(), params.PostalCode, params.Number)
	if err != nil {
		mapErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"street": result.Street, "city": result.City}, "error": nil})
}

func (h *Handler) GetDeliveryPriceEstimate(c *gin.Context, params genshipping.GetDeliveryPriceEstimateParams) {
	options, err := h.getDeliveryEstimate.Execute(c.Request.Context(), appshipping.GetDeliveryPriceEstimateInput{
		ConnectionID: params.ConnectionId.String(), CC: params.Cc, PostalCode: params.PostalCode,
		Number: params.Number, Carrier: shippingDomain.Carrier(params.Carrier),
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	out := make([]genshipping.DeliveryPriceOption, len(options))
	for i := range options {
		out[i] = genshipping.DeliveryPriceOption{
			PriceComment: &options[i].PriceComment, AmountMinorUnits: &options[i].AmountMinorUnits, Currency: &options[i].Currency,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "error": nil})
}

func trackingEventToResponse(e shippingDomain.ShipmentStatusEvent) genshipping.ShipmentTrackingEvent {
	source := string(e.Source)
	label := shippingDomain.MyParcelStatusLabel(e.Status)
	return genshipping.ShipmentTrackingEvent{
		Id: &e.ID, ShipmentId: &e.ShipmentID, Status: &e.Status, StatusMain: &e.StatusMain,
		StatusLabel: &label, Source: (*genshipping.ShipmentTrackingEventSource)(&source), OccurredAt: &e.OccurredAt,
	}
}

func settingsToResponse(s *shippingDomain.ShippingConnectionSettings) *genshipping.ShippingConnectionSettings {
	if s == nil {
		return nil
	}
	carrier, pkgType := int(s.DefaultCarrier), int(s.DefaultPackageType)
	format := genshipping.ShippingConnectionSettingsDefaultLabelFormat(s.DefaultLabelFormat)
	resp := &genshipping.ShippingConnectionSettings{
		ConnectionId: &s.ConnectionID, DefaultCarrier: &carrier, DefaultPackageType: &pkgType,
		DefaultLabelFormat: &format, DefaultSignatureRequired: &s.DefaultSignatureRequired, DefaultOnlyRecipient: &s.DefaultOnlyRecipient,
		InsureByDefault:   &s.InsureByDefault,
		MyparcelAccountId: s.MyParcelAccountID, MyparcelShopId: s.MyParcelShopID, AutoPrintEnabled: &s.AutoPrintEnabled,
		ExportOrderToMyparcel: &s.ExportOrderToMyParcel,
	}
	if s.DefaultLabelSize != "" {
		size := genshipping.ShippingConnectionSettingsDefaultLabelSize(s.DefaultLabelSize)
		resp.DefaultLabelSize = &size
	}
	return resp
}

func overrideToResponse(o shippingDomain.ShippingCarrierOverride) genshipping.ShippingCarrierOverride {
	carrier := int(o.Carrier)
	resp := genshipping.ShippingCarrierOverride{
		ConnectionId: &o.ConnectionID, Carrier: &carrier, SignatureRequired: o.SignatureRequired, OnlyRecipient: o.OnlyRecipient,
		InsureByDefault: o.InsureByDefault, TrackingUrlTemplate: o.TrackingURLTemplate,
	}
	if o.PackageType != nil {
		pt := int(*o.PackageType)
		resp.PackageType = &pt
	}
	return resp
}

func overridesToResponse(overrides []shippingDomain.ShippingCarrierOverride) []genshipping.ShippingCarrierOverride {
	out := make([]genshipping.ShippingCarrierOverride, len(overrides))
	for i, o := range overrides {
		out[i] = overrideToResponse(o)
	}
	return out
}

func shipmentToResponse(s shippingDomain.Shipment) genshipping.Shipment {
	carrier, status := int(s.Carrier), s.Status
	resp := genshipping.Shipment{
		Id: &s.ID, SalesOrderId: &s.SalesOrderID, ConnectionId: &s.ConnectionID, Carrier: &carrier,
		Barcode: &s.Barcode, Status: &status, StatusMain: &s.StatusMain, TrackingUrl: &s.TrackingURL, LabelFormat: &s.LabelFormat,
		CreatedAt: &s.CreatedAt, CancelledAt: s.CancelledAt, CancelNote: &s.CancelNote,
		Recipient: &genshipping.RecipientAddress{
			Person: s.RecipientPerson, Company: &s.RecipientCompany, Street: s.RecipientStreet,
			Number: &s.RecipientNumber, NumberSuffix: &s.RecipientNumberSuffix, PostalCode: &s.RecipientPostalCode,
			City: s.RecipientCity, Cc: s.RecipientCC, Email: &s.RecipientEmail, Phone: &s.RecipientPhone,
		},
	}
	return resp
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
	myParcelFactory := myparcel.NewClientFactory()
	carrierFactory := myparcel.NewCarrierClientFactory()
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	recordActivity := appoms.NewRecordSalesOrderActivityUseCase(db, salesOrderRepo, repositories.NewSalesOrderActivityRepo(db))
	exportMyParcelOrderUC := appshipping.NewExportMyParcelOrderUseCase(db, connRepo, settingsRepo, fulfilmentOrderRepo, salesOrderRepo, lineItemRepo, myParcelFactory, encryptionSecret, recordActivity)

	h := NewHandler(
		appintegrations.NewConnectMyParcelUseCase(db, connRepo, settingsRepo, outboxRepo, encryptionSecret),
		appshipping.NewRegisterMyParcelWebhookUseCase(db, connRepo, myParcelFactory, encryptionSecret, publicAPIBaseURL),
		appshipping.NewGetShippingSettingsUseCase(db, settingsRepo, overrideRepo),
		appshipping.NewUpdateShippingSettingsUseCase(db, settingsRepo),
		appshipping.NewUpsertCarrierOverrideUseCase(db, overrideRepo),
		appshipping.NewCreateShipmentUseCase(db, connRepo, settingsRepo, overrideRepo, shipmentRepo, carrierFactory, encryptionSecret, recordActivity, tenantSettingsRepo, salesOrderRepo, exportMyParcelOrderUC, lineItemRepo, skuRepo),
		appshipping.NewGetShipmentLabelUseCase(db, shipmentRepo, connRepo, settingsRepo, carrierFactory, encryptionSecret),
		appshipping.NewListShipmentsForOrderUseCase(db, shipmentRepo, overrideRepo),
		exportMyParcelOrderUC,
		appshipping.NewListShipmentTrackingEventsUseCase(db, statusEventRepo),
		appshipping.NewRefreshShipmentTrackingUseCase(db, shipmentRepo, statusEventRepo, connRepo, salesOrderRepo, carrierFactory, encryptionSecret),
		appshipping.NewLookupAddressUseCase(pdok.NewClient(pdok.Config{})),
		appshipping.NewCancelShipmentUseCase(db, connRepo, shipmentRepo, carrierFactory, encryptionSecret, recordActivity),
		appshipping.NewGetDeliveryPriceEstimateUseCase(db, connRepo, carrierFactory, encryptionSecret),
	)
	genshipping.RegisterHandlersWithOptions(rg, h, genshipping.GinServerOptions{})
}
