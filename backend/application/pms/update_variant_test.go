package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUpdateVariant_updatesNameAndPositionWithoutTouchingMediaLookup(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Old"}}
	mediaRepo := &stubMediaRepo{}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	v, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large", Position: 1})

	require.NoError(t, err)
	require.Equal(t, "Large", v.Name)
	require.Nil(t, v.FeaturedMediaID)
}

func TestUpdateVariant_setsFeaturedMediaWhenItBelongsToTheVariantsProduct(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Large"}}
	productID := "p1"
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", ProductID: &productID}}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	mediaID := "m1"
	v, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large", FeaturedMediaID: &mediaID})

	require.NoError(t, err)
	require.NotNil(t, v.FeaturedMediaID)
	require.Equal(t, "m1", *v.FeaturedMediaID)
}

func TestUpdateVariant_clearsFeaturedMediaWhenNil(t *testing.T) {
	existingMediaID := "m1"
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1", FeaturedMediaID: &existingMediaID}}
	mediaRepo := &stubMediaRepo{}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	v, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large", FeaturedMediaID: nil})

	require.NoError(t, err)
	require.Nil(t, v.FeaturedMediaID)
}

func TestUpdateVariant_rejectsFeaturedMediaThatDoesNotExist(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	mediaRepo := &stubMediaRepo{findErr: pmsDomain.ErrMediaNotFound}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	mediaID := "missing"
	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large", FeaturedMediaID: &mediaID})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrMediaNotFound))
}

func TestUpdateVariant_rejectsFeaturedMediaBelongingToAnotherProduct(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	otherProductID := "p2"
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", ProductID: &otherProductID}}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	mediaID := "m1"
	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large", FeaturedMediaID: &mediaID})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrFeaturedMediaProductMismatch))
}

func TestUpdateVariant_rejectsFeaturedMediaThatBelongsToAVariantNotAProduct(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1"}}
	variantID := "v2"
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", VariantID: &variantID}}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	mediaID := "m1"
	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large", FeaturedMediaID: &mediaID})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrFeaturedMediaProductMismatch))
}

func TestUpdateVariant_propagatesVariantNotFound(t *testing.T) {
	variantRepo := &stubVariantRepo{findErr: pmsDomain.ErrVariantNotFound}
	mediaRepo := &stubMediaRepo{}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, mediaRepo, &stubProductHistoryRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "missing", Name: "Large"})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVariantNotFound))
}

func TestUpdateVariant_recordsHistoryWhenNameChanges(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Old"}}
	historyRepo := &stubProductHistoryRepo{}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, &stubMediaRepo{}, historyRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large"})

	require.NoError(t, err)
	require.Len(t, historyRepo.created, 1)
	h := historyRepo.created[0]
	require.Equal(t, pmsDomain.ProductHistoryEntityVariant, h.EntityType)
	require.Equal(t, "v1", h.EntityID)
	require.Equal(t, pmsDomain.ProductHistoryFieldName, h.Field)
	require.Equal(t, "Old", h.OldValue)
	require.Equal(t, "Large", h.NewValue)
}

func TestUpdateVariant_doesNotRecordHistoryWhenNameUnchanged(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Same"}}
	historyRepo := &stubProductHistoryRepo{}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, &stubMediaRepo{}, historyRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Same", Position: 2})

	require.NoError(t, err)
	require.Empty(t, historyRepo.created)
}

func TestUpdateVariant_propagatesHistoryRepoError(t *testing.T) {
	variantRepo := &stubVariantRepo{findResult: &pmsDomain.Variant{ID: "v1", ProductID: "p1", Name: "Old"}}
	historyRepo := &stubProductHistoryRepo{createErr: errors.New("db down")}
	uc := apppms.NewUpdateVariantUseCase(newMockGormDB(t), variantRepo, &stubMediaRepo{}, historyRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateVariantInput{ID: "v1", Name: "Large"})

	require.Error(t, err)
}
