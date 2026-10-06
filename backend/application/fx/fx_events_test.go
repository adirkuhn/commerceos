package fx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appfx "github.com/adirkuhn/commerceos/application/fx"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestUpsertExchangeRate_RecordsRateChange(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &fakeExchangeRateRepo{byCurrency: map[string]fxDomain.ExchangeRate{"BRL": {CurrencyCode: "BRL", RateBasisPoints: 55000}}}
	uc := appfx.NewUpsertExchangeRateUseCase(newMockGormDB(t, 1), repo, outbox)

	_, err := uc.Execute(context.Background(), appfx.UpsertExchangeRateInput{CurrencyCode: "BRL", RateBasisPoints: 56000})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "fx.exchange_rate.set", e.Subject)
	assert.Equal(t, "BRL", e.Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"rate_basis_points": {float64(55000), float64(56000)}}, e.Changes())
}

func TestUpsertExchangeRate_NewCurrencyHasNoPreviousRate(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appfx.NewUpsertExchangeRateUseCase(newMockGormDB(t, 1), &fakeExchangeRateRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appfx.UpsertExchangeRateInput{CurrencyCode: "GBP", RateBasisPoints: 8500})

	require.NoError(t, err)
	assert.Equal(t, map[string][2]any{"rate_basis_points": {nil, float64(8500)}}, onlyEvent(t, outbox).Changes())
}

func TestDeleteExchangeRate_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &fakeExchangeRateRepo{byCurrency: map[string]fxDomain.ExchangeRate{"BRL": {CurrencyCode: "BRL", RateBasisPoints: 55000}}}
	uc := appfx.NewDeleteExchangeRateUseCase(newMockGormDB(t, 1), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "BRL"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "fx.exchange_rate.deleted", e.Subject)
	assert.Equal(t, "BRL", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"rate_basis_points": float64(55000)}, e.Data())
}

func TestFXUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}

	_, err := appfx.NewUpsertExchangeRateUseCase(newMockGormDB(t, 1), &fakeExchangeRateRepo{}, failing).
		Execute(context.Background(), appfx.UpsertExchangeRateInput{CurrencyCode: "GBP", RateBasisPoints: 1})
	assert.ErrorIs(t, err, boom)
	assert.ErrorIs(t, appfx.NewDeleteExchangeRateUseCase(newMockGormDB(t, 1), &fakeExchangeRateRepo{}, failing).
		Execute(context.Background(), "GBP"), boom)
}
