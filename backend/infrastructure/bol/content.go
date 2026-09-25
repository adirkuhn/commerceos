package bol

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// --- Product Content API (v10, async -- upload then poll a report) ---

type AttributeValue struct {
	Value string `json:"value"`
	// UnitID is Bol's unit-of-measure code for this value (e.g.
	// "unece.unit.MLT" for ml, "unece.unit.MMT" for mm) -- required for
	// number-typed attributes that have a units list in Bol's data model
	// (Capacity, Width/Height/Length, Weight, Number Pieces In Package).
	// Confirmed against the real spec's AttributeValue schema (unitId,
	// example "mm") and per-attribute unit lists in
	// productdatamodel.s-bol.com/v10/datamodel_v10_en.json. Empty/omitted
	// for text, boolean, and LOV-select attributes, which carry no unit.
	UnitID string `json:"unitId,omitempty"`
}

type ContentAttribute struct {
	ID     string           `json:"id"`
	Values []AttributeValue `json:"values"`
}

// Asset.Labels is an array in Bol's real spec (maxItems 2, example
// ["FRONT"]), not a bare string -- confirmed 2026-08-25 alongside the
// product-content path fix. A bare string previously silently sent the
// wrong JSON shape ("labels":"FRONT" instead of "labels":["FRONT"]), the
// same silent-drop failure mode as a wrong attribute id.
type Asset struct {
	URL    string   `json:"url"`
	Labels []string `json:"labels"`
}

type ProductContentRequest struct {
	Language   string             `json:"language"`
	Attributes []ContentAttribute `json:"attributes"`
	Assets     []Asset            `json:"assets,omitempty"`
}

type processStatusResponse struct {
	ProcessStatusID string `json:"processStatusId"`
	Status          string `json:"status"`
}

type PollOptions struct {
	Interval    time.Duration
	MaxAttempts int
}

func (o PollOptions) withDefaults() PollOptions {
	if o.Interval <= 0 {
		o.Interval = 2 * time.Second
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = 30
	}
	return o
}

// SubmitProductContent uploads new/updated product content, then polls
// process-status until it reaches a terminal status. Bol requires a GPC Code
// + enrichment-level-0/1 mandatory attributes for a genuinely new EAN --
// this client does not validate or invent those; the caller
// (PushProductToBolUseCase) is responsible for populating req.Attributes
// correctly. The exact mandatory-attribute set is category-dependent (Bol's
// "chunks" taxonomy) and was NOT confirmed against a real sandbox during
// this build -- verify before relying on this for a genuinely new category.
//
// Path is POST /content/products (plural). Confirmed 2026-08-25 by
// downloading and inspecting the real, currently-published spec directly --
// api.bol.com/registry/api-definitions/product-content/product-content-v10.yaml,
// operationId post-product-content, path "/retailer/content/products"
// (server root is bare api.bol.com; this client's BaseURL already includes
// /retailer, hence the relative "/content/products" here). Two earlier
// guesses both 404'd in prod: "/product-content" (wrong resource name) and
// then singular "/content/product" (right resource, wrong pluralization --
// taken from a third-party generated client's docs, which was stale/wrong).
// Lesson: don't trust community client docs for this API, download the
// registry spec directly.
//
// Process-status polling hits a genuinely different path root: confirmed
// against api.bol.com/registry/api-definitions/process-status/process-status-v10.yaml,
// path "/shared/process-status/{id}" -- under /shared, not /retailer, so it
// cannot be expressed relative to BaseURL and uses SharedBaseURL instead.
func (c *Client) SubmitProductContent(ctx context.Context, req ProductContentRequest, opts PollOptions) error {
	opts = opts.withDefaults()

	var upload processStatusResponse
	if err := c.doJSON(ctx, http.MethodPost, "/content/products", contentAccept, req, &upload); err != nil {
		return fmt.Errorf("bol: submit product content: %w", err)
	}
	if upload.ProcessStatusID == "" {
		return nil // some responses may be synchronous already-successful, nothing to poll
	}

	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		var report processStatusResponse
		path := "/process-status/" + url.PathEscape(upload.ProcessStatusID)
		if err := c.doJSONWithBase(ctx, c.cfg.SharedBaseURL, http.MethodGet, path, contentAccept, nil, &report); err != nil {
			return fmt.Errorf("bol: poll product content report: %w", err)
		}
		switch strings.ToUpper(report.Status) {
		case "SUCCESS":
			return nil
		case "FAILURE", "TIMEOUT":
			return fmt.Errorf("bol: product content upload %s ended with status %s", upload.ProcessStatusID, report.Status)
		}
		time.Sleep(opts.Interval)
	}
	return fmt.Errorf("bol: product content upload %s did not reach a terminal status after %d attempts", upload.ProcessStatusID, opts.MaxAttempts)
}

// --- Commissions API (v10) ---

type Commission struct {
	EAN         string  `json:"ean"`
	FixedAmount float64 `json:"fixedAmount"`
	Percentage  float64 `json:"percentage"`
}

// GetCommission fetches the live commission for one EAN at unitPrice.
// The exact endpoint path below (GET /commission/{ean}?unit-price=) follows
// Bol's general Retailer API path conventions but was NOT confirmed
// verbatim against the Redoc spec during this build (that page only
// resolved to a Redoc anchor, not a literal path) -- verify against a real
// sandbox call before relying on this in production; if the real path
// differs, only this method's path/query needs to change, nothing else in
// this client depends on it.
func (c *Client) GetCommission(ctx context.Context, ean string, unitPrice float64) (Commission, error) {
	var resp Commission
	path := "/commission/" + url.PathEscape(ean) + "?unit-price=" + strconv.FormatFloat(unitPrice, 'f', 2, 64)
	if err := c.doJSON(ctx, http.MethodGet, path, contentAccept, nil, &resp); err != nil {
		return Commission{}, err
	}
	return resp, nil
}
