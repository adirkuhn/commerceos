package pms

import (
	"context"
	"time"
)

// ProductHistoryEntityType is which kind of pms entity a ProductHistory row
// describes -- the table is shared by products and variants rather than
// having two near-identical tables, since the shape (field/old/new/when) is
// identical either way.
type ProductHistoryEntityType string

const (
	ProductHistoryEntityProduct ProductHistoryEntityType = "product"
	ProductHistoryEntityVariant ProductHistoryEntityType = "variant"
)

// ProductHistoryField is which field changed. Status only ever appears on
// a ProductHistoryEntityProduct row -- variants have no status field.
type ProductHistoryField string

const (
	ProductHistoryFieldName   ProductHistoryField = "name"
	ProductHistoryFieldStatus ProductHistoryField = "status"
)

// ProductHistory is one recorded change to a product or variant's name or
// status -- a lightweight audit trail (v1 scope: name/status only, not a
// generic diff of every field). A row is written only when the field's
// value actually changes.
type ProductHistory struct {
	ID         string
	TenantID   string
	EntityType ProductHistoryEntityType
	EntityID   string
	Field      ProductHistoryField
	OldValue   string
	NewValue   string
	ChangedAt  time.Time
}

type ProductHistoryRepository interface {
	Create(ctx context.Context, h *ProductHistory) error
	ListByEntity(ctx context.Context, entityType ProductHistoryEntityType, entityID string) ([]ProductHistory, error)
}
