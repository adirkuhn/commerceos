package shipping

import "fmt"

// myParcelStatusLabels maps MyParcel's numeric shipment status code (the
// same scale as both the webhook's bare `status` field and TrackTrace's
// `current` field) to a human-readable label -- given directly by the user
// from MyParcel's own status reference, not derived/guessed. 20-29 are not
// part of MyParcel's published range and are intentionally absent.
var myParcelStatusLabels = map[int]string{
	1:  "Pending - concept",
	2:  "Pending - registered",
	3:  "Enroute - handed to carrier",
	4:  "Enroute - sorting",
	5:  "Enroute - distribution",
	6:  "Enroute - customs",
	7:  "Delivered - at recipient",
	8:  "Delivered - ready for pickup",
	9:  "Delivered - package picked up",
	10: "Delivered - return shipment ready for pickup",
	11: "Delivered - return shipment package picked up",
	12: "Printed - letter",
	13: "Credited",
	14: "Printed - digital stamp",
	15: "Printed - external shipment",
	16: "Expired",
	17: "Cancelled",
	18: "Printed - untracked shipment",
	19: "Delivered - at agreed location",
	30: "Inactive - concept",
	31: "Inactive - registered",
	32: "Inactive - enroute - handed to carrier",
	33: "Inactive - enroute - sorting",
	34: "Inactive - enroute - distribution",
	35: "Inactive - enroute - customs",
	36: "Inactive - delivered - at recipient",
	37: "Inactive - delivered - ready for pickup",
	38: "Inactive - delivered - package picked up",
}

// MyParcelStatusLabel returns the human-readable label for one of
// MyParcel's numeric shipment status codes, or a fallback naming the raw
// code if it's outside the published range.
func MyParcelStatusLabel(code int) string {
	if label, ok := myParcelStatusLabels[code]; ok {
		return label
	}
	return fmt.Sprintf("Unknown status (%d)", code)
}
