package pms_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubProductListingRepo struct {
	byProduct []channelsDomain.ProductChannelListing
	listErr   error
	deleted   []string
}

func (s *stubProductListingRepo) Create(_ context.Context, _ *channelsDomain.ProductChannelListing) error {
	return nil
}
func (s *stubProductListingRepo) FindByExternalID(_ context.Context, _, _ string) (*channelsDomain.ProductChannelListing, error) {
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubProductListingRepo) FindByProductAndConnection(_ context.Context, _, _ string) (*channelsDomain.ProductChannelListing, error) {
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubProductListingRepo) FindByProductVariantAndConnection(_ context.Context, _ string, _ *string, _ string) (*channelsDomain.ProductChannelListing, error) {
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubProductListingRepo) ListByConnection(_ context.Context, _ string) ([]channelsDomain.ProductChannelListing, error) {
	return nil, nil
}
func (s *stubProductListingRepo) ListByProduct(_ context.Context, productID string) ([]channelsDomain.ProductChannelListing, error) {
	s.deleted = append(s.deleted, productID)
	return s.byProduct, s.listErr
}
func (s *stubProductListingRepo) ListByProductAndConnection(_ context.Context, _, _ string) ([]channelsDomain.ProductChannelListing, error) {
	return nil, nil
}
func (s *stubProductListingRepo) Update(_ context.Context, _ *channelsDomain.ProductChannelListing) error {
	return nil
}
func (s *stubProductListingRepo) Delete(_ context.Context, _ string) error { return nil }

func TestDeleteProduct_raisesProductDeletedEventWithListingSnapshot(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "prod-1"}}
	listingRepo := &stubProductListingRepo{byProduct: []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "woocommerce", ExternalID: "555"},
	}}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewDeleteProductUseCase(newMockGormDB(t), productRepo, listingRepo, outboxRepo)

	err := uc.Execute(context.Background(), "prod-1")

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.deleted"}, outboxRepo.calls)

	var event events.ProductDeleted
	require.NoError(t, json.Unmarshal(outboxRepo.payloads[0], &event))
	assert.Equal(t, "prod-1", event.ProductID)
	assert.Equal(t, []events.ProductDeletedListing{
		{ConnectionID: "conn-1", Provider: "woocommerce", ExternalID: "555"},
	}, event.Listings)
}

func TestDeleteProduct_raisesEventWithEmptyListingsWhenNeverSynced(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "prod-1"}}
	listingRepo := &stubProductListingRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewDeleteProductUseCase(newMockGormDB(t), productRepo, listingRepo, outboxRepo)

	err := uc.Execute(context.Background(), "prod-1")

	require.NoError(t, err)
	var event events.ProductDeleted
	require.NoError(t, json.Unmarshal(outboxRepo.payloads[0], &event))
	assert.Empty(t, event.Listings)
}

func TestDeleteProduct_snapshotsListingsBeforeDeleting(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "prod-1"}}
	listingRepo := &stubProductListingRepo{byProduct: []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", Provider: "woocommerce", ExternalID: "555"},
	}}
	uc := apppms.NewDeleteProductUseCase(newMockGormDB(t), productRepo, listingRepo, &stubOutboxRepo{})

	err := uc.Execute(context.Background(), "prod-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"prod-1"}, listingRepo.deleted, "must read the product's listings as part of the same delete")
}
