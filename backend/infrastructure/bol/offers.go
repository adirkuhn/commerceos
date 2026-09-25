package bol

import (
	"context"
	"net/http"
	"net/url"
)

// --- Offer API (v11, current -- fully synchronous). Confirmed against
// Bol's real, currently-published Offers API v11.5.0 spec
// (api.bol.com/registry/api-definitions/offers/offers-v11.yaml), found via
// api.bol.com/retailer/public/Retailer-API's own docs navigation, which
// lists "Offers API v11 (current)" as a distinct, separately-versioned API
// from "v10 (deprecated)". This client previously targeted v10 (a wrong
// downgrade from an earlier, also-wrong /v11/offers guess): it accepted
// requests (202 + a real processStatusId) but polling /process-status/{id}
// 404'd in prod -- the deprecated version's documented async completion
// tracking doesn't actually resolve. v11 has no such problem: POST/PATCH
// return the result directly, nothing to poll.
//
// v11's Condition is a discriminated union (NewCondition/
// SecondhandCondition/RefurbishedCondition, keyed by category) -- only
// Category is modeled here since this codebase only ever pushes NEW
// condition products; State/Comment/Grade/Margin would need adding if
// SECONDHAND/REFURBISHED support is ever built.

const offerAccept = "application/vnd.retailer.v11+json"

type BundlePrice struct {
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unitPrice"`
}

type Pricing struct {
	BundlePrices []BundlePrice `json:"bundlePrices"`
}

type Fulfilment struct {
	Method string `json:"method"` // FBR or FBB
	// Schedule is required by Bol when Method is FBR (confirmed via a real
	// 400: "Field 'schedule' is required when 'method' is FBR" -- not
	// visible as schema-required in the static v11.5.0 spec, same kind of
	// business-required-but-not-schema-required gap as economicOperatorId).
	// Enum: BOL_DELIVERY_PROMISE (bol calculates it), MY_DELIVERY_PROMISE
	// (retailer's own profile via the separate Delivery Promise API, not
	// integrated here), SHIPPING_VIA_BOL. Always BOL_DELIVERY_PROMISE for
	// now -- the simplest correct choice with no extra setup required.
	Schedule string `json:"schedule,omitempty"`
	// DeliveryPromise is required by Bol whenever Schedule is
	// BOL_DELIVERY_PROMISE (confirmed via a real 400: "Field
	// 'deliveryPromise' is required when 'method' = FBR and 'schedule' is
	// BOL_DELIVERY_PROMISE" -- again not schema-required in the static
	// spec).
	DeliveryPromise *DeliveryPromise `json:"deliveryPromise,omitempty"`
}

type DeliveryPromise struct {
	MinimumDaysToCustomer int `json:"minimumDaysToCustomer"`
	MaximumDaysToCustomer int `json:"maximumDaysToCustomer"`
}

type Condition struct {
	Category string `json:"category"`
}

type Stock struct {
	Amount            int  `json:"amount"`
	ManagedByRetailer bool `json:"managedByRetailer"`
}

type CreateOfferRequest struct {
	EAN                 string     `json:"ean"`
	UnknownProductTitle string     `json:"unknownProductTitle,omitempty"`
	EconomicOperatorID  string     `json:"economicOperatorId,omitempty"`
	Reference           string     `json:"reference,omitempty"`
	OnHoldByRetailer    bool       `json:"onHoldByRetailer,omitempty"`
	Condition           Condition  `json:"condition"`
	Pricing             Pricing    `json:"pricing"`
	Fulfilment          Fulfilment `json:"fulfilment"`
	Stock               *Stock     `json:"stock,omitempty"`
}

type offerIDResponse struct {
	OfferID string `json:"offerId"`
}

// CreateOffer creates a new Bol offer for req.EAN, returning the new
// offer's id -- synchronous (201 Created) under v11, no polling.
func (c *Client) CreateOffer(ctx context.Context, req CreateOfferRequest) (string, error) {
	var resp offerIDResponse
	if err := c.doJSON(ctx, http.MethodPost, "/offers", offerAccept, req, &resp); err != nil {
		return "", err
	}
	return resp.OfferID, nil
}

// PatchOfferRequest updates an existing offer. Deliberately narrower than
// CreateOfferRequest -- Bol's real PatchOfferRequest schema has no
// condition field at all (an offer's condition can't be changed after
// creation).
type PatchOfferRequest struct {
	EconomicOperatorID string      `json:"economicOperatorId,omitempty"`
	Reference          string      `json:"reference,omitempty"`
	Pricing            *Pricing    `json:"pricing,omitempty"`
	Fulfilment         *Fulfilment `json:"fulfilment,omitempty"`
	Stock              *Stock      `json:"stock,omitempty"`
	// OnHoldByRetailer is a pointer so a call that only wants to flip this
	// flag (SetOfferOnHold) can omit every other field -- omitempty on a nil
	// pointer drops it from the wire payload, matching Bol's real PATCH
	// semantics of "field absent = leave unchanged" (same convention as
	// Pricing/Fulfilment/Stock above).
	OnHoldByRetailer *bool `json:"onHoldByRetailer,omitempty"`
}

// UpdateOffer patches an existing offer's price/stock/etc -- PATCH, not
// PUT (confirmed against the real v11 spec: Bol has no PUT for offers).
// Synchronous (204 No Content).
func (c *Client) UpdateOffer(ctx context.Context, offerID string, req PatchOfferRequest) error {
	return c.doJSON(ctx, http.MethodPatch, "/offers/"+url.PathEscape(offerID), offerAccept, req, nil)
}

// SetOfferOnHold PATCHes only onHoldByRetailer, leaving price/stock/
// fulfilment untouched -- a minimal sibling to UpdateOffer for a caller that
// wants to toggle visibility alone (e.g. gating a listing when the local
// product leaves Active) without recomputing or risking clobbering the
// offer's other fields.
func (c *Client) SetOfferOnHold(ctx context.Context, offerID string, onHold bool) error {
	return c.doJSON(ctx, http.MethodPatch, "/offers/"+url.PathEscape(offerID), offerAccept, PatchOfferRequest{OnHoldByRetailer: &onHold}, nil)
}

type offerListResponse struct {
	Offers []offerSummary `json:"offers"`
}

type offerSummary struct {
	OfferID string `json:"offerId"`
	EAN     string `json:"ean"`
}

// FindOfferByEAN looks up whether an offer already exists on Bol for this
// EAN -- lets PushProductToBolUseCase link and update an offer that
// predates this integration (or survived a lost local listing row)
// instead of blindly POSTing and hitting Bol's 409 Conflict for a
// duplicate offer. Empty string + false (not an error) means no offer
// exists yet for this EAN.
func (c *Client) FindOfferByEAN(ctx context.Context, ean string) (string, bool, error) {
	var resp offerListResponse
	path := "/offers?eans=" + url.QueryEscape(ean)
	if err := c.doJSON(ctx, http.MethodGet, path, offerAccept, nil, &resp); err != nil {
		return "", false, err
	}
	if len(resp.Offers) == 0 {
		return "", false, nil
	}
	return resp.Offers[0].OfferID, true, nil
}

// NotForSaleReasons and its nested types mirror the real v11 spec exactly
// (api.bol.com/registry/api-definitions/offers/offers-v11.yaml,
// operationId get-not-for-sale-reasons, schema NotForSaleReasons) --
// confirmed 2026-08-25 while investigating why a real offer stayed
// "offline" with no error anywhere in this client. Reason is a
// discriminated union in the spec (oneOf ContentReason/PricingReason/
// StockReason/... keyed by "category"); this client only cares about
// Category/Code/Description/MissingAttributes, which every variant has in
// common (MissingAttributes is empty for non-CONTENT reasons), so it's
// modeled as one flat struct rather than a Go type switch -- simpler, and
// nothing here needs the reason-specific fields the other variants might
// carry.
type NotForSaleReasons struct {
	OfferID   string                     `json:"offerId"`
	Countries []NotForSaleReasonsCountry `json:"countries"`
}

type NotForSaleReasonsCountry struct {
	CountryCode string   `json:"countryCode"`
	Reasons     []Reason `json:"reasons"`
}

type Reason struct {
	Category          string             `json:"category"`
	Code              string             `json:"code"`
	Description       string             `json:"description"`
	MissingAttributes []MissingAttribute `json:"missingAttributes,omitempty"`
}

type MissingAttribute struct {
	Name string `json:"name"`
}

// GetNotForSaleReasons reports why an offer is currently not for sale
// (offline), per country -- empty Countries means the offer is for sale
// everywhere. Read-only, safe to call after every push to catch content
// Bol silently rejected (e.g. a still-missing mandatory attribute) that a
// 202-accepted content submission would never otherwise surface.
func (c *Client) GetNotForSaleReasons(ctx context.Context, offerID string) (NotForSaleReasons, error) {
	var resp NotForSaleReasons
	path := "/offers/" + url.PathEscape(offerID) + "/not-for-sale-reasons"
	if err := c.doJSON(ctx, http.MethodGet, path, offerAccept, nil, &resp); err != nil {
		return NotForSaleReasons{}, err
	}
	return resp, nil
}
