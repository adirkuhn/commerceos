package pms_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
)

func TestDeleteSKUCurrencyPriceUseCase_Execute(t *testing.T) {
	repo := &stubSKUCurrencyPriceRepo{}
	uc := apppms.NewDeleteSKUCurrencyPriceUseCase(newMockGormDBN(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "sku-1", "USD")
	require.NoError(t, err)
	require.Equal(t, "sku-1", repo.deletedSKUID)
	require.Equal(t, "USD", repo.deletedCurrency)
}
