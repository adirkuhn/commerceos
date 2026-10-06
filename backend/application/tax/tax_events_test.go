package tax_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apptax "github.com/adirkuhn/commerceos/application/tax"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

// A new default takes the flag off the old one: both are recorded.
func TestCreateTaxClass_RecordsCreatedAndMovedDefault(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubTaxClassRepo{list: []taxDomain.TaxClass{{ID: "old", Name: "Standard", IsDefault: true}}}
	uc := apptax.NewCreateTaxClassUseCase(newMockGormDB(t, 1), repo, outbox)

	_, err := uc.Execute(context.Background(), apptax.CreateTaxClassInput{Name: "Food", IsDefault: true})

	require.NoError(t, err)
	require.Equal(t, []string{"tax.tax_class.updated", "tax.tax_class.created"}, outbox.Subjects())
	assert.Equal(t, "old", outbox.Events[0].Payload["entity_id"])
	assert.Equal(t, map[string][2]any{"is_default": {true, false}}, outbox.Events[0].Changes())
	assert.Equal(t, map[string]any{"reason": "default_moved"}, outbox.Events[0].Data())
	assert.Equal(t, "class-1", outbox.Events[1].Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Food", "is_default": true}, outbox.Events[1].Data())
}

func TestUpdateTaxClass_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Food"}}
	uc := apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), repo, outbox)

	_, err := uc.Execute(context.Background(), apptax.UpdateTaxClassInput{ID: "class-1", Name: "Food (reduced)"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "tax.tax_class.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"name": {"Food", "Food (reduced)"}}, e.Changes())
}

func TestDeleteTaxClass_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1", Name: "Food"}}
	uc := apptax.NewDeleteTaxClassUseCase(newMockGormDB(t, 1), repo, &stubSKUTaxClassChecker{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "class-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "tax.tax_class.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Food"}, e.Data())
}

func TestUpsertTaxRate_RecordsSet(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), &stubTaxRateRepo{}, &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}}, outbox)

	_, err := uc.Execute(context.Background(), apptax.UpsertTaxRateInput{TaxClassID: "class-1", Country: "nl", RateBasisPoints: 900})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "tax.tax_rate.set", e.Subject)
	assert.Equal(t, "rate-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"tax_class_id": "class-1", "country": "NL", "rate_basis_points": float64(900)}, e.Data())
}

func TestDeleteTaxRate_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubTaxRateRepo{findResult: &taxDomain.TaxRate{ID: "rate-1", TaxClassID: "class-1", Country: "NL"}}
	uc := apptax.NewDeleteTaxRateUseCase(newMockGormDB(t, 1), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "rate-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "tax.tax_rate.deleted", e.Subject)
	assert.Equal(t, map[string]any{"tax_class_id": "class-1", "country": "NL"}, e.Data())
}

func TestUpsertExternalTaxClassLink_RecordsSet(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 1), &stubExternalTaxClassLinkRepo{},
		&stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}}, outbox)

	_, err := uc.Execute(context.Background(), apptax.UpsertExternalTaxClassLinkInput{Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "class-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "tax.external_tax_class_link.set", e.Subject)
	assert.Equal(t, "link-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"provider": "woocommerce", "external_slug": "reduced-rate", "tax_class_id": "class-1"}, e.Data())
}

func TestDeleteExternalTaxClassLink_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubExternalTaxClassLinkRepo{findResult: &taxDomain.ExternalTaxClassLink{ID: "link-1", Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "class-1"}}
	uc := apptax.NewDeleteExternalTaxClassLinkUseCase(newMockGormDB(t, 1), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "link-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "tax.external_tax_class_link.deleted", e.Subject)
	assert.Equal(t, map[string]any{"provider": "woocommerce", "external_slug": "reduced-rate", "tax_class_id": "class-1"}, e.Data())
}

func TestTaxUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()
	class := func() *stubTaxClassRepo { return &stubTaxClassRepo{findResult: &taxDomain.TaxClass{ID: "class-1"}} }

	_, err := apptax.NewCreateTaxClassUseCase(newMockGormDB(t, 1), class(), failing).Execute(ctx, apptax.CreateTaxClassInput{Name: "A"})
	assert.ErrorIs(t, err, boom, "create class")
	_, err = apptax.NewUpdateTaxClassUseCase(newMockGormDB(t, 1), class(), failing).Execute(ctx, apptax.UpdateTaxClassInput{ID: "class-1", Name: "B"})
	assert.ErrorIs(t, err, boom, "update class")
	assert.ErrorIs(t, apptax.NewDeleteTaxClassUseCase(newMockGormDB(t, 1), class(), &stubSKUTaxClassChecker{}, failing).Execute(ctx, "class-1"), boom, "delete class")
	_, err = apptax.NewUpsertTaxRateUseCase(newMockGormDB(t, 1), &stubTaxRateRepo{}, class(), failing).Execute(ctx, apptax.UpsertTaxRateInput{TaxClassID: "class-1", Country: "NL"})
	assert.ErrorIs(t, err, boom, "set rate")
	assert.ErrorIs(t, apptax.NewDeleteTaxRateUseCase(newMockGormDB(t, 1), &stubTaxRateRepo{findResult: &taxDomain.TaxRate{ID: "rate-1"}}, failing).Execute(ctx, "rate-1"), boom, "delete rate")
	_, err = apptax.NewUpsertExternalTaxClassLinkUseCase(newMockGormDB(t, 1), &stubExternalTaxClassLinkRepo{}, class(), failing).
		Execute(ctx, apptax.UpsertExternalTaxClassLinkInput{Provider: "woocommerce", TaxClassID: "class-1"})
	assert.ErrorIs(t, err, boom, "set link")
	assert.ErrorIs(t, apptax.NewDeleteExternalTaxClassLinkUseCase(newMockGormDB(t, 1),
		&stubExternalTaxClassLinkRepo{findResult: &taxDomain.ExternalTaxClassLink{ID: "link-1"}}, failing).Execute(ctx, "link-1"), boom, "delete link")
}
