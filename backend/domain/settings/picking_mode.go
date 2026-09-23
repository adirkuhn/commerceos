package settings

// PickingMode is how the picking module works for a tenant. Picking is
// optional (packing is where items are scanned and labels made), so it can
// be as light as a list.
type PickingMode string

const (
	// PickingModeScan: pickers scan each item into a picking session.
	PickingModeScan PickingMode = "scan"
	// PickingModeList: pickers see an order's items as a plain list and
	// mark the whole order picked with one action -- no scanning, no
	// session.
	PickingModeList PickingMode = "list"
)

// ParsePickingMode validates a picking mode; empty means the default, scan.
func ParsePickingMode(s string) (PickingMode, error) {
	switch PickingMode(s) {
	case "", PickingModeScan:
		return PickingModeScan, nil
	case PickingModeList:
		return PickingModeList, nil
	default:
		return "", ErrInvalidPickingMode
	}
}
