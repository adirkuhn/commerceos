package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUpdateMedia_updatesAltText(t *testing.T) {
	productID := "p1"
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", ProductID: &productID, AltText: "Old"}}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "m1", AltText: "New title"})

	require.NoError(t, err)
	require.Equal(t, "New title", m.AltText)
	require.Len(t, mediaRepo.updated, 1)
	require.Equal(t, "New title", mediaRepo.updated[0].AltText)
}

// TestUpdateMedia_raisesProductChangedEventForProductScopedMedia proves an
// alt-text edit on a product-level image reaches Woo -- wooImagesFromMedia
// reads AltText directly, so without this event a lone alt-text edit
// (nothing else on the product changed) never syncs until an unrelated
// product change happens to re-push.
func TestUpdateMedia_raisesProductChangedEventForProductScopedMedia(t *testing.T) {
	productID := "p1"
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", ProductID: &productID}}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "m1", AltText: "New title"})

	require.NoError(t, err)
	require.Equal(t, []string{"pms.product.changed"}, outboxRepo.calls)
}

// TestUpdateMedia_variantOnlyMediaDoesNotRaiseEvent proves a media row
// scoped only to a variant (ProductID nil, used solely as that variant's
// featured-image pick) is a no-op for the event -- mediaRepo.ListByProduct
// filters strictly on product_id, so this row never reaches a Woo push
// regardless of its alt text.
func TestUpdateMedia_variantOnlyMediaDoesNotRaiseEvent(t *testing.T) {
	variantID := "v1"
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", VariantID: &variantID}}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "m1", AltText: "New title"})

	require.NoError(t, err)
	require.Empty(t, outboxRepo.calls)
}

func TestUpdateMedia_propagatesNotFound(t *testing.T) {
	mediaRepo := &stubMediaRepo{findErr: pmsDomain.ErrMediaNotFound}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "missing", AltText: "New title"})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrMediaNotFound))
}

func TestUpdateMedia_propagatesRepoUpdateError(t *testing.T) {
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1"}, updateErr: errors.New("db down")}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "m1", AltText: "New title"})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
