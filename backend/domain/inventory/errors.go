package inventory

import "errors"

var (
	ErrWarehouseNotFound   = errors.New("warehouse not found")
	ErrBatchNotFound       = errors.New("batch not found")
	ErrInvalidQuantity     = errors.New("invalid quantity for movement type")
	ErrInvalidMovementType = errors.New("invalid movement type for this operation")
	ErrBatchHasMovements   = errors.New("batch has stock movements and cannot be deleted")
)
