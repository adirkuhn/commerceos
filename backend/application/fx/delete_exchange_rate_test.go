package fx_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	appfx "github.com/adirkuhn/commerceos/application/fx"
)

func TestDeleteExchangeRateUseCase_Execute(t *testing.T) {
	repo := &fakeExchangeRateRepo{}
	uc := appfx.NewDeleteExchangeRateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "USD")
	require.NoError(t, err)
	require.Equal(t, "USD", repo.deletedCcy)
}
