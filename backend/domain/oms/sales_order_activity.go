package oms

import (
	"context"
	"time"
)

// SalesOrderActivity is one append-only, timestamped entry in a sales
// order's activity log -- replaces ad-hoc use of SalesOrder.CustomerNote
// (a single freeform, overwritable field meant for the customer's own
// note) for anything system-recorded, e.g. MyParcel order export/shipment
// creation/label print and their failures. Tied to both the order and the
// acting user so it can be listed from either side (see
// SalesOrderActivityRepository.ListByUser). Metadata is raw JSON, same
// convention as domain/outbox.Event.Payload -- callers marshal/unmarshal
// at the boundary rather than the domain type modeling every possible
// shape.
type SalesOrderActivity struct {
	ID           string
	TenantID     string
	SalesOrderID string
	// UserID is the external lm2 identity -- no local users table to FK
	// against, same convention as user_permissions/iam_user_groups (see
	// 074_unify_user_id_uuid.sql).
	UserID       string
	ActivityType string
	Summary      string
	Metadata     []byte
	CreatedAt    time.Time
}

type SalesOrderActivityRepository interface {
	Create(ctx context.Context, a *SalesOrderActivity) error
	// ListBySalesOrder returns every activity for the order, newest first
	// -- powers the order detail page's activity timeline.
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]SalesOrderActivity, error)
	// ListByUser returns every activity the given user acted on, across
	// all their orders, newest first -- powers the Users page's per-user
	// activity panel.
	ListByUser(ctx context.Context, userID string) ([]SalesOrderActivity, error)
}

// ActivityTypeMarkedPicked is logged when an order is marked picked in one
// action from the picking module's list mode (MarkOrderPickedUseCase).
const ActivityTypeMarkedPicked = "marked_picked"
