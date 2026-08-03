package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubProductPusher struct {
	calls  []appchannels.PushProductToWooInput
	errFor map[string]error // by ProductID
}

func (s *stubProductPusher) Execute(_ context.Context, in appchannels.PushProductToWooInput) (*channelsDomain.ProductChannelListing, error) {
	s.calls = append(s.calls, in)
	if err, ok := s.errFor[in.ProductID]; ok {
		return nil, err
	}
	return &channelsDomain.ProductChannelListing{ID: "listing-" + in.ProductID, ProductID: in.ProductID, ConnectionID: in.ConnectionID}, nil
}

func TestRunScheduledProductExport_pushesEligibleProducts(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.listResult = []pmsDomain.Product{
		{ID: "p1", Name: "A", HasVariants: false},
		{ID: "p2", Name: "B", HasVariants: false},
	}
	pusher := &stubProductPusher{}

	uc := appchannels.NewRunScheduledProductExportUseCase(newMockGormDB(t, 3), h.productRepo, h.listingRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledProductExportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 2, result.Pushed)
	require.Equal(t, 0, result.Skipped)
	require.Empty(t, result.Errors)
	require.Len(t, pusher.calls, 2)
}

func TestRunScheduledProductExport_skipsProductsWithVariants(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.listResult = []pmsDomain.Product{{ID: "p1", Name: "A", HasVariants: true}}
	pusher := &stubProductPusher{}

	uc := appchannels.NewRunScheduledProductExportUseCase(newMockGormDB(t, 1), h.productRepo, h.listingRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledProductExportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Pushed)
	require.Equal(t, 1, result.Skipped)
	require.Empty(t, pusher.calls)
}

func TestRunScheduledProductExport_skipsAlreadyLinkedProducts(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.listResult = []pmsDomain.Product{{ID: "p1", Name: "A", HasVariants: false}}
	h.listingRepo.byProduct = map[string]channelsDomain.ProductChannelListing{
		"p1": {ID: "listing-1", ProductID: "p1", ConnectionID: "conn-1"},
	}
	pusher := &stubProductPusher{}

	uc := appchannels.NewRunScheduledProductExportUseCase(newMockGormDB(t, 2), h.productRepo, h.listingRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledProductExportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Pushed)
	require.Equal(t, 1, result.Skipped)
	require.Empty(t, pusher.calls)
}

func TestRunScheduledProductExport_recordsErrorForOneProductAndContinues(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.listResult = []pmsDomain.Product{
		{ID: "p1", Name: "A", HasVariants: false},
		{ID: "p2", Name: "B", HasVariants: false},
	}
	pusher := &stubProductPusher{errFor: map[string]error{"p1": errors.New("woo 500")}}

	uc := appchannels.NewRunScheduledProductExportUseCase(newMockGormDB(t, 3), h.productRepo, h.listingRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledProductExportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Pushed)
	require.Len(t, result.Errors, 1)
	require.Len(t, pusher.calls, 2)
}

func TestRunScheduledProductExport_propagatesProductListError(t *testing.T) {
	h := newHarness(nil)
	h.productRepo.listErr = errors.New("db down")
	pusher := &stubProductPusher{}

	uc := appchannels.NewRunScheduledProductExportUseCase(newMockGormDB(t, 1), h.productRepo, h.listingRepo, pusher)

	_, err := uc.Execute(context.Background(), appchannels.RunScheduledProductExportInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
