package tools

import (
	"context"
	"encoding/json"

	"github.com/adirkuhn/commerceos/domain/assistant"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

type shipmentsLister interface {
	Execute(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error)
}

type trackingEventsLister interface {
	Execute(ctx context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error)
}

type bolConfirmationsLister interface {
	Execute(ctx context.Context, salesOrderID string) ([]channelsDomain.BolShipmentConfirmation, error)
}

// GetOrderShipmentsTool answers "where is my package": every shipment
// (label) of an order with its tracking and carrier milestones, plus what
// Bol was told for a Bol order.
type GetOrderShipmentsTool struct {
	shipments shipmentsLister
	tracking  trackingEventsLister
	bol       bolConfirmationsLister
	resolve   orderResolver
}

func NewGetOrderShipmentsTool(shipments shipmentsLister, tracking trackingEventsLister, bol bolConfirmationsLister, search globalSearcher) *GetOrderShipmentsTool {
	return &GetOrderShipmentsTool{shipments: shipments, tracking: tracking, bol: bol, resolve: orderResolver{search: search}}
}

func (t *GetOrderShipmentsTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_order_shipments",
		Description: "Shipping and tracking of one sales order (id or order number): each label with carrier, tracking " +
			"number and link, latest status, when the label was made, first carrier scan and delivery -- and, for a Bol " +
			"order, whether Bol was told it shipped. Use for \"where is my order/package\" questions.",
		Parameters: object(map[string]any{"order": str("The order id (uuid) or order number.")}, "order"),
	}
}

type shipmentView struct {
	Carrier            string `json:"carrier"`
	TrackingNumber     string `json:"tracking_number,omitempty"`
	TrackingURL        string `json:"tracking_url,omitempty"`
	Status             string `json:"status,omitempty"`
	LabelCreatedAt     string `json:"label_created_at"`
	FirstCarrierScanAt string `json:"first_carrier_scan_at,omitempty"`
	DeliveredAt        string `json:"delivered_at,omitempty"`
	// Active is false for a cancelled label (e.g. wrong carrier picked).
	Active     bool   `json:"active"`
	CancelNote string `json:"cancel_note,omitempty"`
}

type bolConfirmationView struct {
	Action          string `json:"action"`
	Status          string `json:"status"`
	TransporterCode string `json:"transporter_code,omitempty"`
	TrackAndTrace   string `json:"track_and_trace,omitempty"`
	Error           string `json:"error,omitempty"`
	SentAt          string `json:"sent_at"`
	ResolvedAt      string `json:"resolved_at,omitempty"`
}

func (t *GetOrderShipmentsTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Order string `json:"order"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("order", args.Order); err != nil {
		return "", err
	}
	id, err := t.resolve.id(ctx, args.Order)
	if err != nil {
		return "", err
	}
	shipments, err := t.shipments.Execute(ctx, id)
	if err != nil {
		return "", err
	}
	out := struct {
		Shipments        []shipmentView        `json:"shipments"`
		BolConfirmations []bolConfirmationView `json:"bol_confirmations,omitempty"`
	}{Shipments: make([]shipmentView, 0, len(shipments))}
	for _, s := range shipments {
		events, err := t.tracking.Execute(ctx, s.ID)
		if err != nil {
			return "", err
		}
		out.Shipments = append(out.Shipments, toShipmentView(s, events))
	}
	confirmations, err := t.bol.Execute(ctx, id)
	if err != nil {
		return "", err
	}
	for _, c := range confirmations {
		out.BolConfirmations = append(out.BolConfirmations, bolConfirmationView{
			Action: string(c.Action), Status: string(c.Status), TransporterCode: c.TransporterCode, TrackAndTrace: c.TrackAndTrace,
			Error: firstNonEmpty(c.FailureReason, c.ErrorMessage), SentAt: formatTime(c.CreatedAt), ResolvedAt: formatTimePtr(c.ResolvedAt),
		})
	}
	return toJSON(out)
}

// toShipmentView takes the milestones from the status history: the first
// status after "registered" is the carrier's first scan.
func toShipmentView(s shippingDomain.Shipment, events []shippingDomain.ShipmentStatusEvent) shipmentView {
	v := shipmentView{
		Carrier: s.Carrier.String(), TrackingNumber: s.Barcode, TrackingURL: s.TrackingURL, Status: s.StatusMain,
		LabelCreatedAt: formatTime(s.CreatedAt), Active: s.IsActive(), CancelNote: s.CancelNote,
	}
	for _, e := range events {
		if v.FirstCarrierScanAt == "" && e.StatusMain != "" && e.StatusMain != shippingDomain.StatusMainRegistered {
			v.FirstCarrierScanAt = formatTime(e.OccurredAt)
		}
		if v.DeliveredAt == "" && e.StatusMain == shippingDomain.StatusMainDelivered {
			v.DeliveredAt = formatTime(e.OccurredAt)
		}
	}
	return v
}
