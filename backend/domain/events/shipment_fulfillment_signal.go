package events

// ShipmentFulfillmentSignal is raised whenever inbound MyParcel activity
// implies a SalesOrder might need to advance -- both its Shipments API
// (shipment_status_change webhook, and ReconcileShipmentsUseCase's poll of
// the same API) and its separate Orders/Fulfilment API (order_status_change
// webhook's "processed" status) raise this same event instead of each
// deciding inline what it means for the order (see
// application/shipping.ApplyShipmentFulfillmentSignalUseCase, the one place
// that decision is made, consumed via
// infrastructure/consumers.ShipmentFulfillmentSignalConsumer).
type ShipmentFulfillmentSignal struct {
	Base
	TenantID     string `json:"tenant_id"`
	SalesOrderID string `json:"sales_order_id"`
	// Kind is one of the ShipmentFulfillmentSignalKind* constants below.
	Kind string `json:"kind"`
	// StatusMain/StatusCode are MyParcel's Shipments-API tracktrace fields --
	// only meaningful when Kind is ShipmentFulfillmentSignalKindShipmentStatus;
	// zero-value for ShipmentFulfillmentSignalKindOrderProcessed (the Orders
	// API carries no comparable status string).
	StatusMain string `json:"status_main,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

const (
	// ShipmentFulfillmentSignalKindShipmentStatus is MyParcel's Shipments
	// API status report -- raised by HandleShipmentWebhookUseCase (webhook)
	// and ReconcileShipmentsUseCase (poll).
	ShipmentFulfillmentSignalKindShipmentStatus = "shipment_status"
	// ShipmentFulfillmentSignalKindOrderProcessed is MyParcel's separate
	// Orders/Fulfilment API's "processed" webhook (HandleOrderStatusWebhookUseCase)
	// -- its own unambiguous "MyParcel fulfilled this order" signal,
	// independent of any Shipments-API status or local Fulfilled state.
	ShipmentFulfillmentSignalKindOrderProcessed = "order_processed"
)

func (ShipmentFulfillmentSignal) EventName() string { return "shipping.shipment.fulfillment_signal" }
