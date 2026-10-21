package postnl

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// Label creates the label without pre-announcing it; Confirm pre-announces
// it. Unlike the one-call labelconfirm endpoint (base Benelux parcels
// only), this pair also covers PostNL's EU products. Both run back to back,
// so a created shipment is always pre-announced.
const (
	labelPath   = "/shipment/delivery/v4/label"
	confirmPath = "/shipment/delivery/v4/confirm"
)

// domesticCountries get PostNL's domestic products and services.
var domesticCountries = map[string]bool{"NL": true, "BE": true}

// euCountries is where PostNL's "Parcel EU" products deliver (developer
// portal, Products & services Dutch International).
var euCountries = map[string]bool{
	"AT": true, "BG": true, "HR": true, "CY": true, "DK": true, "EE": true, "FI": true, "FR": true, "DE": true,
	"SI": true, "GR": true, "HU": true, "IE": true, "IT": true, "LV": true, "LT": true, "LU": true, "MT": true,
	"CZ": true, "PL": true, "PT": true, "RO": true, "SK": true, "ES": true, "SE": true,
}

// EU bundles: Track & Trace, insured up to 50 euro, insured up to 500 euro.
const (
	bundleTrackTrace  = "track_trace"
	bundleInsured     = "insured"
	bundleInsuredPlus = "insured_plus"

	insuredMaxMinorUnits = 5000
)

// shipmentTypes maps this codebase's PackageType values (MyParcel's
// numbering, see domain/shipping.PackageType) to PostNL's shipmentType.
// Package types missing here (digital stamp, pallet) have no PostNL
// equivalent.
var shipmentTypes = map[int]string{
	1: "parcel",    // package
	2: "letterbox", // mailbox package
	3: "letter",    // letter
	6: "packet",    // small package
	7: "letter",    // envelope
}

var labelContentTypes = map[string]string{
	"pdf": "application/pdf",
	"zpl": "application/vnd.shipment_label+zpl",
}

type address struct {
	CompanyName         string `json:"companyName,omitempty"`
	Street              string `json:"street,omitempty"`
	HouseNumber         string `json:"houseNumber,omitempty"`
	HouseNumberAddition string `json:"houseNumberAddition,omitempty"`
	PostalCode          string `json:"postalCode,omitempty"`
	City                string `json:"city,omitempty"`
	CountryIso          string `json:"countryIso"`
}

type contact struct {
	Email        string `json:"email,omitempty"`
	FirstName    string `json:"firstName,omitempty"`
	LastName     string `json:"lastName,omitempty"`
	MobileNumber string `json:"mobileNumber,omitempty"`
}

type shipmentRequest struct {
	Receiver                  receiver                   `json:"receiver"`
	Sender                    sender                     `json:"sender"`
	Items                     []shipmentItem             `json:"items"`
	ShipmentType              string                     `json:"shipmentType"`
	Services                  *services                  `json:"services,omitempty"`
	InternationalShipmentData *internationalShipmentData `json:"internationalShipmentData,omitempty"`
	LabelSettings             *labelSettings             `json:"labelSettings,omitempty"`
}

type internationalShipmentData struct {
	Bundle         string `json:"bundle"`
	PartnerID      string `json:"partnerId,omitempty"`
	PartnerBarcode string `json:"partnerBarcode,omitempty"`
}

type receiver struct {
	Type    string   `json:"type"`
	Address address  `json:"address"`
	Contact *contact `json:"contact,omitempty"`
}

type sender struct {
	CustomerNumber             string   `json:"customerNumber"`
	CustomerCode               string   `json:"customerCode"`
	Address                    address  `json:"address"`
	UndeliverableReturnAddress *address `json:"undeliverableReturnAddress,omitempty"`
	Contact                    *contact `json:"contact,omitempty"`
}

type shipmentItem struct {
	Barcode            string             `json:"barcode,omitempty"`
	Dimensions         *dimensions        `json:"dimensions,omitempty"`
	CustomerReferences customerReferences `json:"customerReferences"`
}

type dimensions struct {
	Weight int `json:"weight"`
}

type customerReferences struct {
	ShipmentReference string `json:"shipmentReference"`
	ShipmentRemark    string `json:"shipmentRemark,omitempty"`
}

type services struct {
	DeliveryConfirmation string   `json:"deliveryConfirmation,omitempty"`
	StatedAddressOnly    bool     `json:"statedAddressOnly,omitempty"`
	InsuredValue         *float64 `json:"insuredValue,omitempty"`
}

type labelSettings struct {
	OutputType string `json:"outputType"`
}

type shipmentResponse struct {
	Items []struct {
		Barcode        string `json:"barcode"`
		PartnerID      string `json:"partnerId"`
		PartnerBarcode string `json:"partnerBarcode"`
		Labels         []struct {
			Label      string `json:"label"`
			OutputType string `json:"outputType"`
		} `json:"labels"`
	} `json:"items"`
}

// CreateShipment creates the label (Label) and pre-announces it (Confirm)
// for a one-parcel shipment. Its ExternalShipmentID is our own random
// reference (sent as customerReferences.shipmentReference), which
// GetTrackTrace looks the status up by. If Confirm fails the shipment
// counts as failed: the label exists but PostNL doesn't know it, and an
// unscanned label costs nothing.
func (c *Client) CreateShipment(ctx context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
	req, format, err := c.shipmentRequestOf(in)
	if err != nil {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: %w", err)
	}

	var labelResp shipmentResponse
	if err := c.do(ctx, http.MethodPost, labelPath, req, &labelResp); err != nil {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create label: %w", err)
	}
	if len(labelResp.Items) == 0 {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create label: response has no items")
	}
	item := labelResp.Items[0]

	// Confirm mirrors the Label request, minus labelSettings, plus what
	// the Label response assigned.
	confirm := req
	confirm.LabelSettings = nil
	confirm.Items = []shipmentItem{req.Items[0]}
	confirm.Items[0].Barcode = item.Barcode
	if req.InternationalShipmentData != nil {
		isd := *req.InternationalShipmentData
		isd.PartnerID, isd.PartnerBarcode = item.PartnerID, item.PartnerBarcode
		confirm.InternationalShipmentData = &isd
	}
	var confirmResp shipmentResponse
	if err := c.do(ctx, http.MethodPost, confirmPath, confirm, &confirmResp); err != nil {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: confirm shipment: %w", err)
	}

	result := integrationsDomain.CarrierShipmentResult{
		ExternalShipmentID: req.Items[0].CustomerReferences.ShipmentReference, Barcode: item.Barcode,
	}
	if len(item.Labels) > 0 && item.Labels[0].Label != "" {
		data, err := base64.StdEncoding.DecodeString(item.Labels[0].Label)
		if err != nil {
			return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create label: decode label: %w", err)
		}
		result.Label = &integrationsDomain.LabelResult{Data: data, ContentType: labelContentTypes[format]}
	}
	return result, nil
}

// shipmentRequestOf builds the Label request and the label format used.
func (c *Client) shipmentRequestOf(in integrationsDomain.CreateCarrierShipmentRequest) (shipmentRequest, string, error) {
	if in.Sender == nil {
		return shipmentRequest{}, "", integrationsDomain.ErrCarrierSenderAddressMissing
	}
	shipmentType, ok := shipmentTypes[in.Options.PackageType]
	if !ok {
		return shipmentRequest{}, "", fmt.Errorf("package type %d: %w", in.Options.PackageType, integrationsDomain.ErrCarrierOperationNotSupported)
	}
	cc := strings.ToUpper(in.Recipient.CC)
	domestic, eu := domesticCountries[cc], euCountries[cc]
	if !domestic && !eu {
		return shipmentRequest{}, "", fmt.Errorf("country %q: %w", cc, integrationsDomain.ErrCarrierDestinationNotSupported)
	}
	reference, err := newReference()
	if err != nil {
		return shipmentRequest{}, "", err
	}
	format := in.LabelFormat
	if _, ok := labelContentTypes[format]; !ok {
		format = "pdf"
	}

	req := shipmentRequest{
		Receiver: receiverOf(in.Recipient),
		Sender: sender{
			CustomerNumber: c.customerNumber, CustomerCode: c.customerCode,
			Address: addressOf(*in.Sender, in.Sender.Company), Contact: contactOf(*in.Sender, false),
		},
		Items:         []shipmentItem{{CustomerReferences: customerReferences{ShipmentReference: reference, ShipmentRemark: in.OrderReference}}},
		ShipmentType:  shipmentType,
		LabelSettings: &labelSettings{OutputType: format},
	}
	if in.ReturnAddress != nil {
		ret := addressOf(*in.ReturnAddress, in.ReturnAddress.Company)
		req.Sender.UndeliverableReturnAddress = &ret
	}
	if domestic {
		req.Services = servicesOf(in.Options)
	} else {
		req.InternationalShipmentData = &internationalShipmentData{Bundle: euBundle(in.Options.InsuranceAmountMinorUnits)}
	}
	if in.WeightGrams != nil && *in.WeightGrams > 0 {
		req.Items[0].Dimensions = &dimensions{Weight: *in.WeightGrams}
	}
	return req, format, nil
}

// euBundle is the EU product for an insured amount: insured covers up to
// 50 euro, insured_plus up to 500 euro (the most PostNL offers).
func euBundle(insuranceMinor *int) string {
	switch {
	case insuranceMinor == nil:
		return bundleTrackTrace
	case *insuranceMinor <= insuredMaxMinorUnits:
		return bundleInsured
	default:
		return bundleInsuredPlus
	}
}

func receiverOf(r integrationsDomain.CarrierAddress) receiver {
	kind := "consumer"
	if r.Company != "" {
		kind = "business"
	}
	return receiver{Type: kind, Address: addressOf(r, r.Company), Contact: contactOf(r, true)}
}

func addressOf(a integrationsDomain.CarrierAddress, company string) address {
	return address{
		CompanyName: company, Street: a.Street, HouseNumber: a.Number, HouseNumberAddition: a.NumberSuffix,
		PostalCode: strings.ReplaceAll(a.PostalCode, " ", ""), City: a.City, CountryIso: strings.ToUpper(a.CC),
	}
}

// contactOf is the person PostNL sends delivery notifications to; the
// sender's contact is just its email.
func contactOf(a integrationsDomain.CarrierAddress, withPerson bool) *contact {
	c := contact{Email: a.Email}
	if withPerson {
		c.FirstName, c.LastName = splitName(a.Person)
		if len(a.Phone) >= 7 {
			c.MobileNumber = a.Phone
		}
	}
	if c == (contact{}) {
		return nil
	}
	return &c
}

// splitName splits "Jane van Dijk" into "Jane" + "van Dijk". A single
// name is also the last name -- PostNL's EU products require one.
func splitName(person string) (string, string) {
	first, last, _ := strings.Cut(strings.TrimSpace(person), " ")
	if last = strings.TrimSpace(last); last == "" {
		last = first
	}
	return first, last
}

func servicesOf(o integrationsDomain.CarrierShipmentOptions) *services {
	s := services{StatedAddressOnly: o.OnlyRecipient}
	if o.SignatureRequired {
		s.DeliveryConfirmation = "signature"
	}
	if o.InsuranceAmountMinorUnits != nil {
		v := float64(*o.InsuranceAmountMinorUnits) / 100
		s.InsuredValue = &v
	}
	if s == (services{}) {
		return nil
	}
	return &s
}

// newReference is a 32-character unique shipment reference.
func newReference() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate reference: %w", err)
	}
	return hex.EncodeToString(b), nil
}
