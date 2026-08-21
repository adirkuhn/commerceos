package scheduler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	appscheduler "github.com/adirkuhn/commerceos/application/scheduler"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// stubResyncListingRepo implements only what ListAll needs -- every other
// method is unreachable from RunResyncProductsToChannelsUseCase.Run.
type stubResyncListingRepo struct {
	all []channelsDomain.ProductChannelListing
}

func (s *stubResyncListingRepo) Create(context.Context, *channelsDomain.ProductChannelListing) error {
	return nil
}
func (s *stubResyncListingRepo) FindByExternalID(context.Context, string, string) (*channelsDomain.ProductChannelListing, error) {
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubResyncListingRepo) FindByProductAndConnection(context.Context, string, string) (*channelsDomain.ProductChannelListing, error) {
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubResyncListingRepo) FindByProductVariantAndConnection(context.Context, string, *string, string) (*channelsDomain.ProductChannelListing, error) {
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubResyncListingRepo) ListByConnection(context.Context, string) ([]channelsDomain.ProductChannelListing, error) {
	return nil, nil
}
func (s *stubResyncListingRepo) ListByProduct(context.Context, string) ([]channelsDomain.ProductChannelListing, error) {
	return nil, nil
}
func (s *stubResyncListingRepo) ListByProductAndConnection(context.Context, string, string) ([]channelsDomain.ProductChannelListing, error) {
	return nil, nil
}
func (s *stubResyncListingRepo) Update(context.Context, *channelsDomain.ProductChannelListing) error {
	return nil
}
func (s *stubResyncListingRepo) Delete(context.Context, string) error { return nil }
func (s *stubResyncListingRepo) ListAll(context.Context) ([]channelsDomain.ProductChannelListing, error) {
	return s.all, nil
}

// stubEmptyVariantRepo always reports zero variants -- keeps
// ResyncProductToChannelsUseCase's stock loop a no-op so this test can
// focus purely on RunResyncProductsToChannelsUseCase's own dedup/dispatch
// logic.
type stubEmptyVariantRepo struct{}

func (stubEmptyVariantRepo) Create(context.Context, *pmsDomain.Variant) error { return nil }
func (stubEmptyVariantRepo) FindByID(context.Context, string) (*pmsDomain.Variant, error) {
	return nil, pmsDomain.ErrVariantNotFound
}
func (stubEmptyVariantRepo) ListByProduct(context.Context, string) ([]pmsDomain.Variant, error) {
	return nil, nil
}
func (stubEmptyVariantRepo) Update(context.Context, *pmsDomain.Variant) error { return nil }
func (stubEmptyVariantRepo) Delete(context.Context, string) error             { return nil }

type stubUnreachableSKURepo struct{ pmsDomain.SKURepository }

// stubTenantRecordingFieldsSyncer records both the product id and which
// tenant the call ran under (via tenant.IDFromContext) -- proving
// RunResyncProductsToChannelsUseCase dispatches each product under its own
// tenant's context, not the cross-tenant super_admin one used to list them.
type stubTenantRecordingFieldsSyncer struct {
	calls []tenantProductCall
}

type tenantProductCall struct {
	TenantID  string
	ProductID string
}

func (s *stubTenantRecordingFieldsSyncer) Execute(ctx context.Context, productID string) error {
	s.calls = append(s.calls, tenantProductCall{TenantID: tenant.IDFromContext(ctx), ProductID: productID})
	return nil
}

func TestRunResyncProductsToChannels_dedupesByTenantAndProductAcrossMultipleListings(t *testing.T) {
	listingRepo := &stubResyncListingRepo{all: []channelsDomain.ProductChannelListing{
		{TenantID: "tenant-a", ProductID: "product-1", ConnectionID: "conn-1"},
		{TenantID: "tenant-a", ProductID: "product-1", ConnectionID: "conn-2"}, // same product, 2nd channel -- must not double-resync
		{TenantID: "tenant-a", ProductID: "product-2", ConnectionID: "conn-1"},
		{TenantID: "tenant-b", ProductID: "product-1", ConnectionID: "conn-3"}, // different tenant, same product id
	}}
	fieldsSyncer := &stubTenantRecordingFieldsSyncer{}
	resyncUC := appchannels.NewResyncProductToChannelsUseCase(newMockGormDBN(t, 4), stubEmptyVariantRepo{}, stubUnreachableSKURepo{}, fieldsSyncer, nil)
	uc := appscheduler.NewRunResyncProductsToChannelsUseCase(newMockGormDBN(t, 1), listingRepo, resyncUC)

	err := uc.Run(context.Background())

	require.NoError(t, err)
	require.Len(t, fieldsSyncer.calls, 3, "3 distinct (tenant, product) pairs despite 4 listing rows")
	require.Contains(t, fieldsSyncer.calls, tenantProductCall{TenantID: "tenant-a", ProductID: "product-1"})
	require.Contains(t, fieldsSyncer.calls, tenantProductCall{TenantID: "tenant-a", ProductID: "product-2"})
	require.Contains(t, fieldsSyncer.calls, tenantProductCall{TenantID: "tenant-b", ProductID: "product-1"})
}

func TestRunResyncProductsToChannels_noListingsIsNoOp(t *testing.T) {
	listingRepo := &stubResyncListingRepo{}
	fieldsSyncer := &stubTenantRecordingFieldsSyncer{}
	resyncUC := appchannels.NewResyncProductToChannelsUseCase(newMockGormDBN(t, 0), stubEmptyVariantRepo{}, stubUnreachableSKURepo{}, fieldsSyncer, nil)
	uc := appscheduler.NewRunResyncProductsToChannelsUseCase(newMockGormDBN(t, 1), listingRepo, resyncUC)

	err := uc.Run(context.Background())

	require.NoError(t, err)
	require.Empty(t, fieldsSyncer.calls)
}
