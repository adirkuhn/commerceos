package channels

import "context"

type ProductChannelListingRepository interface {
	Create(ctx context.Context, l *ProductChannelListing) error
	FindByExternalID(ctx context.Context, connectionID, externalID string) (*ProductChannelListing, error)
	FindByProductAndConnection(ctx context.Context, productID, connectionID string) (*ProductChannelListing, error)
	ListByConnection(ctx context.Context, connectionID string) ([]ProductChannelListing, error)
	Update(ctx context.Context, l *ProductChannelListing) error
}
