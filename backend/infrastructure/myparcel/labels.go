package myparcel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type LabelResult struct {
	Data        []byte
	ContentType string
}

type TrackTraceResult struct {
	StatusCode  int
	StatusMain  string
	TrackingURL string
}

// GetLabel fetches a shipment's label. format is "pdf" or "zpl" -- see
// domain/shipping.LabelFormatPDF/LabelFormatZPL -- the response Content-Type
// (Accept header), fixed at shipment-creation time. size is the physical
// page format for this fetch (domain/shipping.LabelSizeA4/LabelSizeA6),
// sent as the `format` query param -- see the package doc comment's third
// flagged item; unconfirmed against a real sandbox response. Empty size
// omits the query param, leaving MyParcel's own default.
func (c *Client) GetLabel(ctx context.Context, myParcelShipmentID int64, format, size string) (LabelResult, error) {
	accept := "application/pdf"
	if format == "zpl" {
		accept = "application/vnd.shipment_label+zpl"
	}

	url := c.cfg.BaseURL + "/shipment_labels/" + strconv.FormatInt(myParcelShipmentID, 10)
	if size != "" {
		url += "?format=" + size
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return LabelResult{}, fmt.Errorf("myparcel: build label request: %w", err)
	}
	req.Header.Set(authHeader, c.authValue())
	req.Header.Set("Accept", accept)

	resp, err := c.http.Do(req)
	if err != nil {
		return LabelResult{}, fmt.Errorf("myparcel: label request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LabelResult{}, &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return LabelResult{Data: body, ContentType: resp.Header.Get("Content-Type")}, nil
}

type trackTraceEnvelope struct {
	Data struct {
		TrackTraces []struct {
			Status struct {
				Current int    `json:"current"`
				Main    string `json:"main"`
			} `json:"status"`
			// LinkConsumerPortal is the carrier-specific tracking page
			// (e.g. PostNL's own tracktrace URL) MyParcel's docs say is
			// meant to be shared with the end customer -- confirmed against
			// developer.myparcel.nl/api-reference/06.shipments.html's own
			// example response.
			LinkConsumerPortal string `json:"link_consumer_portal"`
		} `json:"tracktraces"`
	} `json:"data"`
}

// GetTrackTrace fetches a shipment's current tracking status. See the
// package doc comment and domain/shipping.Shipment.IsTerminal for the
// status caveats.
func (c *Client) GetTrackTrace(ctx context.Context, myParcelShipmentID int64) (TrackTraceResult, error) {
	var resp trackTraceEnvelope
	path := "/tracktraces/" + strconv.FormatInt(myParcelShipmentID, 10)
	if err := c.doJSON(ctx, http.MethodGet, path, "application/json", nil, &resp); err != nil {
		return TrackTraceResult{}, err
	}
	if len(resp.Data.TrackTraces) == 0 {
		return TrackTraceResult{}, fmt.Errorf("myparcel: get tracktrace: empty response")
	}
	tt := resp.Data.TrackTraces[0]
	return TrackTraceResult{StatusCode: tt.Status.Current, StatusMain: tt.Status.Main, TrackingURL: tt.LinkConsumerPortal}, nil
}
