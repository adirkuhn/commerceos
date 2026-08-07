package suppliers

import "context"

type SupplierRepository interface {
	Create(ctx context.Context, s *Supplier) error
	FindByID(ctx context.Context, id string) (*Supplier, error)
	// FindByName looks up a supplier by case-insensitive, trimmed exact name
	// match. Returns ErrSupplierNotFound if none matches.
	FindByName(ctx context.Context, name string) (*Supplier, error)
	List(ctx context.Context) ([]Supplier, error)
	Update(ctx context.Context, s *Supplier) error
	Delete(ctx context.Context, id string) error
}
