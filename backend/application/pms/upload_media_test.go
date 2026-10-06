package pms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestUploadMedia_rejectsWhenNoTargetGiven(t *testing.T) {
	productRepo := &stubMediaProductRepo{}
	variantRepo := &stubMediaVariantRepo{}
	mediaRepo := &stubMediaRepo{}
	fileStorage := &stubFileStorage{}
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 0), productRepo, variantRepo, mediaRepo, fileStorage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppms.UploadMediaInput{
		Filename: "photo.jpg", ContentType: "image/jpeg", Data: strings.NewReader("bytes"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrMediaTargetRequired))
	require.Empty(t, mediaRepo.attached)
}

func TestUploadMedia_rejectsUnsupportedContentType(t *testing.T) {
	productID := "product-1"
	productRepo := &stubMediaProductRepo{product: &pmsDomain.Product{ID: productID}}
	variantRepo := &stubMediaVariantRepo{}
	mediaRepo := &stubMediaRepo{}
	fileStorage := &stubFileStorage{}
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 0), productRepo, variantRepo, mediaRepo, fileStorage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppms.UploadMediaInput{
		ProductID: &productID, Filename: "doc.pdf", ContentType: "application/pdf", Data: strings.NewReader("bytes"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrUnsupportedMediaType))
	require.Empty(t, fileStorage.savedFilename)
}

func TestUploadMedia_propagatesProductNotFoundWithoutTouchingStorage(t *testing.T) {
	productID := "missing-product"
	productRepo := &stubMediaProductRepo{findErr: pmsDomain.ErrProductNotFound}
	variantRepo := &stubMediaVariantRepo{}
	mediaRepo := &stubMediaRepo{}
	fileStorage := &stubFileStorage{}
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 1), productRepo, variantRepo, mediaRepo, fileStorage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppms.UploadMediaInput{
		ProductID: &productID, Filename: "photo.jpg", ContentType: "image/jpeg", Data: strings.NewReader("bytes"),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrProductNotFound))
	require.Empty(t, fileStorage.savedFilename)
}

func TestUploadMedia_uploadsForAProductAndAttachesTheResultingURL(t *testing.T) {
	productID := "product-1"
	productRepo := &stubMediaProductRepo{product: &pmsDomain.Product{ID: productID, TenantID: "tenant-1"}}
	variantRepo := &stubMediaVariantRepo{}
	mediaRepo := &stubMediaRepo{byProduct: []pmsDomain.Media{{ID: "existing-1"}}}
	fileStorage := &stubFileStorage{returnedURL: "https://api.example.com/uploads/tenant-1/abc.jpg"}
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 2), productRepo, variantRepo, mediaRepo, fileStorage, &eventrecordtest.Outbox{})

	media, err := uc.Execute(context.Background(), apppms.UploadMediaInput{
		ProductID: &productID, Filename: "photo.jpg", ContentType: "image/jpeg",
		Data: strings.NewReader("fake-bytes"), AltText: "Front view",
	})

	require.NoError(t, err)
	require.Equal(t, "https://api.example.com/uploads/tenant-1/abc.jpg", media.URL)
	require.Equal(t, "Front view", media.AltText)
	require.Equal(t, pmsDomain.MediaTypeImage, media.Type)
	require.Equal(t, &productID, media.ProductID)
	require.Nil(t, media.VariantID)
	require.Equal(t, 1, media.Position, "must be positioned after the one existing image")

	require.Equal(t, "photo.jpg", fileStorage.savedFilename)
	require.Equal(t, "fake-bytes", string(fileStorage.savedData))
	require.Len(t, mediaRepo.attached, 1)
}

func TestUploadMedia_uploadsForAVariant(t *testing.T) {
	variantID := "variant-1"
	productRepo := &stubMediaProductRepo{}
	variantRepo := &stubMediaVariantRepo{variant: &pmsDomain.Variant{ID: variantID, TenantID: "tenant-1"}}
	mediaRepo := &stubMediaRepo{}
	fileStorage := &stubFileStorage{}
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 2), productRepo, variantRepo, mediaRepo, fileStorage, &eventrecordtest.Outbox{})

	media, err := uc.Execute(context.Background(), apppms.UploadMediaInput{
		VariantID: &variantID, Filename: "swatch.png", ContentType: "image/png", Data: strings.NewReader("bytes"),
	})

	require.NoError(t, err)
	require.Equal(t, &variantID, media.VariantID)
	require.Nil(t, media.ProductID)
}

func TestUploadMedia_doesNotAttachWhenStorageSaveFails(t *testing.T) {
	productID := "product-1"
	productRepo := &stubMediaProductRepo{product: &pmsDomain.Product{ID: productID}}
	variantRepo := &stubMediaVariantRepo{}
	mediaRepo := &stubMediaRepo{}
	fileStorage := &stubFileStorage{saveErr: errors.New("disk full")}
	uc := apppms.NewUploadMediaUseCase(newMockGormDBN(t, 1), productRepo, variantRepo, mediaRepo, fileStorage, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), apppms.UploadMediaInput{
		ProductID: &productID, Filename: "photo.jpg", ContentType: "image/jpeg", Data: strings.NewReader("bytes"),
	})

	require.Error(t, err)
	require.Empty(t, mediaRepo.attached)
}
