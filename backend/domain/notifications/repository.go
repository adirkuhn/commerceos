package notifications

import "context"

type Repository interface {
	Create(ctx context.Context, n *Notification) error
	// List returns the most recent notifications, newest first.
	List(ctx context.Context, limit int) ([]Notification, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context) error
}
