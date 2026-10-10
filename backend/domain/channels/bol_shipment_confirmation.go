package channels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidBolConfirmationStatus   = errors.New("invalid Bol shipment confirmation status")
	ErrBolConfirmationAlreadyResolved = errors.New("Bol shipment confirmation already has a final result")
	ErrBolConfirmationNotFound        = errors.New("Bol shipment confirmation not found")
)

// BolConfirmationStatus mirrors Bol's own process-status values -- a
// shipment sent to Bol is processed asynchronously, and these are the
// states that processing can be in.
type BolConfirmationStatus string

const (
	BolConfirmationPending BolConfirmationStatus = "pending"
	BolConfirmationSuccess BolConfirmationStatus = "success"
	BolConfirmationFailure BolConfirmationStatus = "failure"
	BolConfirmationTimeout BolConfirmationStatus = "timeout"
)

var bolConfirmationStatuses = []BolConfirmationStatus{
	BolConfirmationPending, BolConfirmationSuccess, BolConfirmationFailure, BolConfirmationTimeout,
}

// ParseBolConfirmationStatus reads a status as Bol reports it (PENDING,
// SUCCESS, FAILURE, TIMEOUT), case-insensitively.
func ParseBolConfirmationStatus(s string) (BolConfirmationStatus, error) {
	for _, status := range bolConfirmationStatuses {
		if strings.EqualFold(s, string(status)) {
			return status, nil
		}
	}
	return "", fmt.Errorf("%w: %q", ErrInvalidBolConfirmationStatus, s)
}

// BolConfirmationAction is which Bol request a confirmation made.
type BolConfirmationAction string

const (
	// BolActionCreateShipment marks order items shipped (POST /shipments).
	BolActionCreateShipment BolConfirmationAction = "create_shipment"
	// BolActionUpdateTransport replaces the transporter + track & trace of
	// a shipment Bol already has (PUT /transports/{id}) -- used when the
	// parcel Bol was told about was cancelled and replaced by another.
	BolActionUpdateTransport BolConfirmationAction = "update_transport"
)

// BolShipmentConfirmation records one attempt to tell Bol that a Bol order
// shipped (POST /retailer/shipments) and what Bol made of it. Bol accepts
// the request straight away but only decides later, so a confirmation
// starts Pending with Bol's ProcessStatusID and is resolved once Bol has a
// final answer. A confirmation that could not even be sent (e.g. a carrier
// Bol has no code for) is stored already Failed, with the reason, so the
// order page can tell the merchant to confirm it by hand.
type BolShipmentConfirmation struct {
	ID              string
	TenantID        string
	SalesOrderID    string
	ConnectionID    string
	ShipmentID      string
	ExternalOrderID string
	Action          BolConfirmationAction
	// TransportID is Bol's transport updated by a BolActionUpdateTransport
	// confirmation; empty for BolActionCreateShipment.
	TransportID     string
	ProcessStatusID string
	TransporterCode string
	TrackAndTrace   string
	Status          BolConfirmationStatus
	// FailureReason says why a confirmation failed before or when reaching
	// Bol (see BolFailure*), so the admin can explain it in the reader's
	// language; empty otherwise. ErrorMessage is the English (or Bol's
	// own) wording.
	FailureReason string
	ErrorMessage  string
	CreatedAt     time.Time
	ResolvedAt    *time.Time
}

// Resolve records Bol's final answer. Only a pending confirmation can be
// resolved, and only to a final status.
func (c *BolShipmentConfirmation) Resolve(status BolConfirmationStatus, errorMessage string, now time.Time) error {
	if c.Status != BolConfirmationPending {
		return ErrBolConfirmationAlreadyResolved
	}
	if status == BolConfirmationPending {
		return fmt.Errorf("%w: cannot resolve to pending", ErrInvalidBolConfirmationStatus)
	}
	c.Status = status
	c.ErrorMessage = errorMessage
	c.ResolvedAt = &now
	return nil
}

// ToldBol is true while Bol has (or is processing) this shipment's
// details -- pending or success.
func (c BolShipmentConfirmation) ToldBol() bool {
	return c.Status == BolConfirmationPending || c.Status == BolConfirmationSuccess
}

// NeedsManualAction is true when Bol did not accept the shipment, so the
// order is still open on Bol until the merchant confirms it there.
func (c BolShipmentConfirmation) NeedsManualAction() bool {
	return c.Status == BolConfirmationFailure || c.Status == BolConfirmationTimeout
}

// BolShipmentConfirmationRepository stores BolShipmentConfirmations.
// Tenant-scoped (RLS); ListPending is called with bypass_rls by the
// cross-tenant polling job.
type BolShipmentConfirmationRepository interface {
	Create(ctx context.Context, c *BolShipmentConfirmation) error
	// ListBySalesOrder returns the order's confirmations, newest first.
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]BolShipmentConfirmation, error)
	// ListPending returns up to limit pending confirmations, oldest first.
	ListPending(ctx context.Context, limit int) ([]BolShipmentConfirmation, error)
	// SaveResolution persists Status, ErrorMessage and ResolvedAt.
	SaveResolution(ctx context.Context, c *BolShipmentConfirmation) error
}

// Why a Bol shipment confirmation failed -- BolShipmentConfirmation.FailureReason.
const (
	// BolFailureNoTransporterCode: Bol has no transporter code for the
	// shipment's carrier, so nothing could be sent.
	BolFailureNoTransporterCode = "no_transporter_code"
	// BolFailureNoBolShipment: a replaced label needed correcting, but Bol
	// has no shipment for the order to correct.
	BolFailureNoBolShipment = "no_bol_shipment"
	// BolFailureRejected: Bol rejected the request; ErrorMessage is Bol's
	// own reason.
	BolFailureRejected = "rejected"
)
