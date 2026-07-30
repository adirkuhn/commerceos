package inventory

import "errors"

var (
	ErrWarehouseNotFound = errors.New("warehouse not found")
	ErrBatchNotFound     = errors.New("batch not found")
	ErrInvalidQuantity   = errors.New("invalid quantity for movement type")
)
