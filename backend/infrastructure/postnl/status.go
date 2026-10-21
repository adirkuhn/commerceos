package postnl

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// sharedStatus is a status on this codebase's shared tracking scale --
// MyParcel's numeric codes and status.main values (see
// domain/shipping.MyParcelStatusLabel and StatusMainDelivered), which the
// order timeline, IsCarrierConfirmed and delivery completion all read.
type sharedStatus struct {
	code int
	main string
}

var (
	statusRegistered      = sharedStatus{2, "registered"}
	statusHandedToCarrier = sharedStatus{3, "handed_to_carrier"}
	statusSorting         = sharedStatus{4, "sorting"}
	statusDistribution    = sharedStatus{5, "distribution"}
	statusCustoms         = sharedStatus{6, "customs"}
	statusDelivered       = sharedStatus{7, "delivered"}
	statusReadyForPickup  = sharedStatus{8, "delivered"}
	statusPickedUp        = sharedStatus{9, "delivered"}
)

// postnlStatuses maps PostNL's T&T status codes (developer portal,
// reference data "T&T status codes") onto the shared scale. Codes not
// listed (99 "not applicable") count as registered.
var postnlStatuses = map[string]sharedStatus{
	"1": statusRegistered, "4": statusRegistered, "13": statusRegistered, "14": statusRegistered,
	"2": statusHandedToCarrier, "3": statusHandedToCarrier,
	"5": statusSorting, "6": statusSorting, "15": statusSorting, "16": statusSorting, "18": statusSorting, "19": statusSorting, "21": statusSorting,
	"7": statusDistribution, "8": statusDistribution, "17": statusDistribution,
	"9": statusCustoms, "20": statusCustoms,
	"11": statusDelivered,
	"12": statusReadyForPickup,
	"22": statusPickedUp, "23": statusPickedUp,
}

type statusResponse struct {
	CurrentStatus *struct {
		Shipment struct {
			MainBarcode string `json:"MainBarcode"`
			Status      struct {
				StatusCode string `json:"StatusCode"`
			} `json:"Status"`
		} `json:"Shipment"`
	} `json:"CurrentStatus"`
}

func (c *Client) status(ctx context.Context, reference string) (statusResponse, error) {
	q := url.Values{"customerCode": {c.customerCode}, "customerNumber": {c.customerNumber}}
	var resp statusResponse
	err := c.do(ctx, http.MethodGet, "/shipment/v2/status/reference/"+url.PathEscape(reference)+"?"+q.Encode(), nil, &resp)
	return resp, err
}

// GetTrackTrace is the shipment's current status, looked up by the
// reference CreateShipment returned. A shipment PostNL hasn't processed
// yet answers with only a warning -- still registered.
func (c *Client) GetTrackTrace(ctx context.Context, reference string) (integrationsDomain.TrackTraceResult, error) {
	resp, err := c.status(ctx, reference)
	if err != nil {
		return integrationsDomain.TrackTraceResult{}, fmt.Errorf("postnl: get status: %w", err)
	}
	s := statusRegistered
	if resp.CurrentStatus != nil {
		if mapped, ok := postnlStatuses[resp.CurrentStatus.Shipment.Status.StatusCode]; ok {
			s = mapped
		}
	}
	return integrationsDomain.TrackTraceResult{StatusCode: s.code, StatusMain: s.main}, nil
}

func (c *Client) GetShipmentBarcode(ctx context.Context, reference string) (string, error) {
	resp, err := c.status(ctx, reference)
	if err != nil {
		return "", fmt.Errorf("postnl: get barcode: %w", err)
	}
	if resp.CurrentStatus == nil {
		return "", nil
	}
	return resp.CurrentStatus.Shipment.MainBarcode, nil
}
