package customers

import "context"

type CustomerRepository interface {
	Create(ctx context.Context, c *Customer) error
	FindByID(ctx context.Context, id string) (*Customer, error)
	List(ctx context.Context) ([]Customer, error)
	Update(ctx context.Context, c *Customer) error
	Delete(ctx context.Context, id string) error
}
