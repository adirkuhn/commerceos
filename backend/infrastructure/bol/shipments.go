package bol

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// --- Shipments API (Retailer API v10) ---
// Confirmed against Bol's published Retailer API v10 spec: POST /shipments
// takes a ShipmentRequest (orderItems, shipmentReference, transport
// {transporterCode, trackAndTrace}) and answers 202 with a ProcessStatus.
// PUT /transports/{id} (ChangeTransportRequest) corrects a shipment's
// transporter + track & trace; its transport id comes from
// GET /shipments?order-id=. Bol only decides later -- GetProcessStatus
// polls the shared /process-status/{id} resource (see
// Config.SharedBaseURL).

type shipmentOrderItem struct {
	OrderItemID string `json:"orderItemId"`
	Quantity    int    `json:"quantity"`
}

type shipmentTransport struct {
	TransporterCode string `json:"transporterCode"`
	TrackAndTrace   string `json:"trackAndTrace,omitempty"`
}

type shipmentRequest struct {
	OrderItems        []shipmentOrderItem `json:"orderItems"`
	ShipmentReference string              `json:"shipmentReference,omitempty"`
	Transport         shipmentTransport   `json:"transport"`
}

type processStatusReport struct {
	ProcessStatusID string `json:"processStatusId"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"errorMessage"`
}

func (r processStatusReport) toDomain() integrationsDomain.BolProcessStatus {
	return integrationsDomain.BolProcessStatus{ID: r.ProcessStatusID, Status: r.Status, ErrorMessage: r.ErrorMessage}
}

func (a *Adapter) CreateShipment(ctx context.Context, in integrationsDomain.BolShipmentInput) (integrationsDomain.BolProcessStatus, error) {
	req := shipmentRequest{
		ShipmentReference: in.ShipmentReference,
		Transport:         shipmentTransport{TransporterCode: in.TransporterCode, TrackAndTrace: in.TrackAndTrace},
	}
	for _, it := range in.OrderItems {
		req.OrderItems = append(req.OrderItems, shipmentOrderItem{OrderItemID: it.OrderItemID, Quantity: it.Quantity})
	}
	var report processStatusReport
	if err := a.client.doJSON(ctx, http.MethodPost, "/shipments", contentAccept, req, &report); err != nil {
		return integrationsDomain.BolProcessStatus{}, classifyRejection(fmt.Errorf("bol: create shipment: %w", err))
	}
	return report.toDomain(), nil
}

type shipmentsListResponse struct {
	Shipments []struct {
		Transport struct {
			TransportID string `json:"transportId"`
		} `json:"transport"`
	} `json:"shipments"`
}

func (a *Adapter) ListShipmentTransportIDs(ctx context.Context, orderID string) ([]string, error) {
	var resp shipmentsListResponse
	if err := a.client.doJSON(ctx, http.MethodGet, "/shipments?order-id="+url.QueryEscape(orderID), contentAccept, nil, &resp); err != nil {
		return nil, fmt.Errorf("bol: list shipments for order %s: %w", orderID, err)
	}
	ids := make([]string, 0, len(resp.Shipments))
	for _, s := range resp.Shipments {
		if s.Transport.TransportID != "" {
			ids = append(ids, s.Transport.TransportID)
		}
	}
	return ids, nil
}

func (a *Adapter) UpdateTransport(ctx context.Context, transportID string, in integrationsDomain.BolTransportInput) (integrationsDomain.BolProcessStatus, error) {
	var report processStatusReport
	path := "/transports/" + url.PathEscape(transportID)
	body := shipmentTransport{TransporterCode: in.TransporterCode, TrackAndTrace: in.TrackAndTrace}
	if err := a.client.doJSON(ctx, http.MethodPut, path, contentAccept, body, &report); err != nil {
		return integrationsDomain.BolProcessStatus{}, classifyRejection(fmt.Errorf("bol: update transport %s: %w", transportID, err))
	}
	return report.toDomain(), nil
}

func (a *Adapter) GetProcessStatus(ctx context.Context, processStatusID string) (integrationsDomain.BolProcessStatus, error) {
	var report processStatusReport
	path := "/process-status/" + url.PathEscape(processStatusID)
	if err := a.client.doJSONWithBase(ctx, a.client.cfg.SharedBaseURL, http.MethodGet, path, contentAccept, nil, &report); err != nil {
		return integrationsDomain.BolProcessStatus{}, fmt.Errorf("bol: get process status %s: %w", processStatusID, err)
	}
	return report.toDomain(), nil
}

// classifyRejection marks a 4xx Bol answer as permanent
// (ErrBolRequestRejected). Auth failures (401/403) and rate limiting (429)
// stay plain errors: those are worth retrying later.
func classifyRejection(err error) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		return err
	}
	if apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		return fmt.Errorf("%w: %w", integrationsDomain.ErrBolRequestRejected, err)
	}
	return err
}
