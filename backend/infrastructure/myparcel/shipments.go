package myparcel

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// --- Shipments ---

// Recipient/Options/ShipmentInput/ShipmentResult/LabelResult/TrackTraceResult
// are this package's own request/response shapes -- kept separate from
// domain/integrations' MyParcelClient port types so domain/application never
// import infrastructure types (CLAUDE.md's layering rule). adapter.go maps
// between the two at the boundary.

type Recipient struct {
	Person       string
	Company      string
	Street       string
	Number       string
	NumberSuffix string
	PostalCode   string
	City         string
	CC           string
	Email        string
	Phone        string
}

type Options struct {
	PackageType       int
	SignatureRequired bool
	OnlyRecipient     bool
	// InsuranceAmountMinorUnits 0 means not insured.
	InsuranceAmountMinorUnits int
	// InsuranceCurrency is required by MyParcel whenever insured (see
	// insuranceOption). Empty falls back to "EUR" -- CreateShipment's own
	// default when a caller doesn't have a tenant currency to hand (e.g.
	// never configured Settings -> Localization -> Default Currency).
	InsuranceCurrency string
}

// defaultInsuranceCurrency is CreateShipment's fallback when the caller
// passes no InsuranceCurrency -- matches this client's original hardcoded
// behavior for a tenant with no Settings -> Localization -> Default
// Currency configured yet (domain/settings.TenantSettings' zero value).
const defaultInsuranceCurrency = "EUR"

type ShipmentInput struct {
	Recipient      Recipient
	Carrier        int
	Options        Options
	OrderReference string
	// WeightGrams -- see domain/integrations.CreateShipmentInput's doc
	// comment. Nil omits physical_properties entirely.
	WeightGrams *int
}

type ShipmentResult struct {
	ShipmentID int64
	Barcode    string
}

type recipient struct {
	Person       string `json:"person"`
	Company      string `json:"company,omitempty"`
	Street       string `json:"street"`
	Number       string `json:"number,omitempty"`
	NumberSuffix string `json:"number_suffix,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	City         string `json:"city"`
	CC           string `json:"cc"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

type insuranceOption struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

// Signature/OnlyRecipient are int (0/1), not bool -- confirmed against a
// real MyParcel 422: "Boolean value found, but a number is required...
// Does not have a value in the enumeration [0,1]". Go's json package
// marshals a bool field as JSON true/false regardless of omitempty;
// MyParcel's schema wants the numeric form.
type shipmentOptions struct {
	PackageType   int              `json:"package_type"`
	Signature     int              `json:"signature,omitempty"`
	OnlyRecipient int              `json:"only_recipient,omitempty"`
	Insurance     *insuranceOption `json:"insurance,omitempty"`
}

func boolToMyParcelInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// physicalProperties carries a shipment's weight -- confirmed against
// 07.shipment-related-object-definitions.html: shipments[].physical_properties.weight,
// an integer number of grams, required for non-EU and digital-stamp
// shipments (harmless to include otherwise).
type physicalProperties struct {
	Weight int `json:"weight"`
}

type createShipmentRequest struct {
	Carrier            int                 `json:"carrier"`
	Recipient          recipient           `json:"recipient"`
	Options            shipmentOptions     `json:"options"`
	PhysicalProperties *physicalProperties `json:"physical_properties,omitempty"`
	// ReferenceIdentifier round-trips our own SalesOrder.ID -- see the
	// package doc comment's second flagged item on the exact field name.
	ReferenceIdentifier string `json:"reference_identifier,omitempty"`
}

// shipmentIdsEnvelope is POST /shipments' real response envelope --
// data.ids[]{id,reference_identifier} -- see the package doc comment for
// why this isn't data.shipments[]{id,barcode} (the original, wrong guess).
type shipmentIdsEnvelope struct {
	Data struct {
		IDs []struct {
			ID                  int64  `json:"id"`
			ReferenceIdentifier string `json:"reference_identifier"`
		} `json:"ids"`
	} `json:"data"`
}

// shipmentSearchEnvelope is GET /shipments/{id}'s response envelope --
// data.search_results.shipments[], confirmed against
// 06.shipments.html's search example. Used only by getShipmentByID's
// post-create barcode fetch; other fields in the real response (recipient,
// options, price, etc) are intentionally not modeled here since nothing
// needs them yet.
type shipmentSearchEnvelope struct {
	Data struct {
		SearchResults struct {
			Shipments []struct {
				ID      int64  `json:"id"`
				Barcode string `json:"barcode"`
				Status  int    `json:"status"`
			} `json:"shipments"`
		} `json:"search_results"`
	} `json:"data"`
}

// getShipmentByID fetches a just-created shipment's barcode (and current
// status) -- CreateShipment's own response never includes them (see the
// package doc comment). Errors here are handled by the caller as non-fatal:
// the shipment id is already confirmed to exist on MyParcel's side by this
// point, so treating a barcode-fetch failure as a full CreateShipment
// failure would make the caller re-POST on retry and create a genuine
// second MyParcel shipment -- exactly the duplicate-creation bug this fix
// is for. A missing barcode gets backfilled later by
// ReconcileShipmentsUseCase's poll (ShipmentRepository.FindStaleNonTerminal).
// GetShipmentBarcode fetches a shipment's barcode via GET /shipments/{id}.
// See domain/integrations.MyParcelClient.GetShipmentBarcode's doc for why
// this exists separately from GetTrackTrace.
func (c *Client) GetShipmentBarcode(ctx context.Context, myParcelShipmentID int64) (string, error) {
	barcode, _, err := c.getShipmentByID(ctx, myParcelShipmentID)
	return barcode, err
}

func (c *Client) getShipmentByID(ctx context.Context, id int64) (barcode string, status int, err error) {
	var resp shipmentSearchEnvelope
	path := "/shipments/" + strconv.FormatInt(id, 10)
	if err := c.doJSON(ctx, http.MethodGet, path, shipmentAccept, nil, &resp); err != nil {
		return "", 0, err
	}
	if len(resp.Data.SearchResults.Shipments) == 0 {
		return "", 0, fmt.Errorf("myparcel: get shipment %d: empty response", id)
	}
	s := resp.Data.SearchResults.Shipments[0]
	return s.Barcode, s.Status, nil
}

// DeleteShipment cancels a shipment on MyParcel's side (DELETE
// /shipments/{id}). Per MyParcel's own docs (developer.myparcel.nl/
// api-reference/06.shipments.html) this only succeeds while the shipment is
// still status 1 ("pending - concept"), i.e. before a label has been
// created -- since this client creates the label as part of CreateShipment,
// MyParcel is expected to reject most real calls here with a 422/*APIError.
// Callers must treat that as an informative outcome, not a hard failure --
// see application/shipping.CancelShipmentUseCase.
func (c *Client) DeleteShipment(ctx context.Context, myParcelShipmentID int64) error {
	path := "/shipments/" + strconv.FormatInt(myParcelShipmentID, 10)
	return c.doJSON(ctx, http.MethodDelete, path, shipmentAccept, nil, nil)
}

// CreateShipment creates one MyParcel shipment. MyParcel's create-shipment
// endpoint accepts an array of shipments in one call; this client always
// sends exactly one and returns its result.
func (c *Client) CreateShipment(ctx context.Context, in ShipmentInput) (ShipmentResult, error) {
	req := struct {
		Data struct {
			Shipments []createShipmentRequest `json:"shipments"`
		} `json:"data"`
	}{}
	options := shipmentOptions{
		PackageType: in.Options.PackageType, Signature: boolToMyParcelInt(in.Options.SignatureRequired), OnlyRecipient: boolToMyParcelInt(in.Options.OnlyRecipient),
	}
	if in.Options.InsuranceAmountMinorUnits > 0 {
		currency := in.Options.InsuranceCurrency
		if currency == "" {
			currency = defaultInsuranceCurrency
		}
		options.Insurance = &insuranceOption{Amount: in.Options.InsuranceAmountMinorUnits, Currency: currency}
	}
	var physProps *physicalProperties
	if in.WeightGrams != nil {
		physProps = &physicalProperties{Weight: *in.WeightGrams}
	}
	req.Data.Shipments = []createShipmentRequest{{
		Carrier: in.Carrier,
		Recipient: recipient{
			Person: in.Recipient.Person, Company: in.Recipient.Company,
			Street: in.Recipient.Street, Number: in.Recipient.Number, NumberSuffix: in.Recipient.NumberSuffix,
			PostalCode: in.Recipient.PostalCode, City: in.Recipient.City, CC: in.Recipient.CC,
			Email: in.Recipient.Email, Phone: in.Recipient.Phone,
		},
		Options:             options,
		PhysicalProperties:  physProps,
		ReferenceIdentifier: in.OrderReference,
	}}

	var resp shipmentIdsEnvelope
	if err := c.doJSON(ctx, http.MethodPost, "/shipments", shipmentAccept, req, &resp); err != nil {
		return ShipmentResult{}, err
	}
	if len(resp.Data.IDs) == 0 {
		return ShipmentResult{}, fmt.Errorf("myparcel: create shipment: empty response")
	}
	id := resp.Data.IDs[0].ID

	barcode, _, err := c.getShipmentByID(ctx, id)
	if err != nil {
		// The shipment id is confirmed real -- see getShipmentByID's doc
		// comment for why this doesn't fail the whole create.
		return ShipmentResult{ShipmentID: id}, nil
	}
	return ShipmentResult{ShipmentID: id, Barcode: barcode}, nil
}
