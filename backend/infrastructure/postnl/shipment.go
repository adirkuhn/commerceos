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

const labelconfirmPath = "/shipment/delivery/v4/labelconfirm"

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
	Receiver      receiver       `json:"receiver"`
	Sender        sender         `json:"sender"`
	Items         []shipmentItem `json:"items"`
	ShipmentType  string         `json:"shipmentType"`
	Services      *services      `json:"services,omitempty"`
	LabelSettings labelSettings  `json:"labelSettings"`
}

type receiver struct {
	Type    string   `json:"type"`
	Address address  `json:"address"`
	Contact *contact `json:"contact,omitempty"`
}

type sender struct {
	CustomerNumber string   `json:"customerNumber"`
	CustomerCode   string   `json:"customerCode"`
	Address        address  `json:"address"`
	Contact        *contact `json:"contact,omitempty"`
}

type shipmentItem struct {
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
		Barcode string `json:"barcode"`
		Labels  []struct {
			Label      string `json:"label"`
			OutputType string `json:"outputType"`
		} `json:"labels"`
	} `json:"items"`
}

// CreateShipment creates and pre-announces a one-parcel shipment. Its
// ExternalShipmentID is our own random reference (sent as
// customerReferences.shipmentReference), which GetTrackTrace looks the
// status up by.
func (c *Client) CreateShipment(ctx context.Context, in integrationsDomain.CreateCarrierShipmentRequest) (integrationsDomain.CarrierShipmentResult, error) {
	if in.Sender == nil {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: %w", integrationsDomain.ErrCarrierSenderAddressMissing)
	}
	shipmentType, ok := shipmentTypes[in.Options.PackageType]
	if !ok {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: package type %d: %w", in.Options.PackageType, integrationsDomain.ErrCarrierOperationNotSupported)
	}
	reference, err := newReference()
	if err != nil {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: %w", err)
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
		Services:      servicesOf(in.Options),
		LabelSettings: labelSettings{OutputType: format},
	}
	if in.WeightGrams != nil && *in.WeightGrams > 0 {
		req.Items[0].Dimensions = &dimensions{Weight: *in.WeightGrams}
	}

	var resp shipmentResponse
	if err := c.do(ctx, http.MethodPost, labelconfirmPath, req, &resp); err != nil {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: %w", err)
	}
	if len(resp.Items) == 0 {
		return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: response has no items")
	}
	item := resp.Items[0]
	result := integrationsDomain.CarrierShipmentResult{ExternalShipmentID: reference, Barcode: item.Barcode}
	if len(item.Labels) > 0 && item.Labels[0].Label != "" {
		data, err := base64.StdEncoding.DecodeString(item.Labels[0].Label)
		if err != nil {
			return integrationsDomain.CarrierShipmentResult{}, fmt.Errorf("postnl: create shipment: decode label: %w", err)
		}
		result.Label = &integrationsDomain.LabelResult{Data: data, ContentType: labelContentTypes[format]}
	}
	return result, nil
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

// splitName splits "Jane van Dijk" into "Jane" + "van Dijk".
func splitName(person string) (string, string) {
	first, last, _ := strings.Cut(strings.TrimSpace(person), " ")
	return first, strings.TrimSpace(last)
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
