package pms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestDeleteMedia_deletesDBRowThenTheStoredFile(t *testing.T) {
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "media-1", URL: "https://api.example.com/uploads/t1/x.jpg"}}
	fileStorage := &stubFileStorage{}
	uc := apppms.NewDeleteMediaUseCase(newMockGormDBN(t, 1), mediaRepo, fileStorage)

	err := uc.Execute(context.Background(), "media-1")

	require.NoError(t, err)
	require.Equal(t, []string{"media-1"}, mediaRepo.deleted)
	require.Equal(t, "https://api.example.com/uploads/t1/x.jpg", fileStorage.deletedURL)
}

func TestDeleteMedia_propagatesNotFoundWithoutTouchingStorage(t *testing.T) {
	mediaRepo := &stubMediaRepo{}
	fileStorage := &stubFileStorage{}
	uc := apppms.NewDeleteMediaUseCase(newMockGormDBN(t, 1), mediaRepo, fileStorage)

	err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrMediaNotFound))
	require.Empty(t, fileStorage.deletedURL)
}

func TestDeleteMedia_surfacesStorageDeleteFailureAfterDBRowIsAlreadyGone(t *testing.T) {
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "media-1", URL: "https://api.example.com/uploads/t1/x.jpg"}}
	fileStorage := &stubFileStorage{deleteErr: errors.New("permission denied")}
	uc := apppms.NewDeleteMediaUseCase(newMockGormDBN(t, 1), mediaRepo, fileStorage)

	err := uc.Execute(context.Background(), "media-1")

	require.Error(t, err)
	require.Equal(t, []string{"media-1"}, mediaRepo.deleted, "DB row is removed even if the file cleanup afterward fails")
}
