package customers

import "context"

type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	FindByID(ctx context.Context, id string) (*Customer, error)
	// FindByWooCustomerID looks up a customer already linked to a Woo
	// customer -- used by order import to resolve/attach the buyer without
	// creating a duplicate. Nil (not an error) when none exists, same
	// convention as oms.SalesOrderRepository.FindByExternalID.
	FindByWooCustomerID(ctx context.Context, wooCustomerID int64) (*Customer, error)
	List(ctx context.Context) ([]Customer, error)
	Update(ctx context.Context, c *Customer) error
	Delete(ctx context.Context, id string) error
}
