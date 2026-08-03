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
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1", AltText: "Old"}}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo)

	m, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "m1", AltText: "New title"})

	require.NoError(t, err)
	require.Equal(t, "New title", m.AltText)
	require.Len(t, mediaRepo.updated, 1)
	require.Equal(t, "New title", mediaRepo.updated[0].AltText)
}

func TestUpdateMedia_propagatesNotFound(t *testing.T) {
	mediaRepo := &stubMediaRepo{findErr: pmsDomain.ErrMediaNotFound}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "missing", AltText: "New title"})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrMediaNotFound))
}

func TestUpdateMedia_propagatesRepoUpdateError(t *testing.T) {
	mediaRepo := &stubMediaRepo{findResult: &pmsDomain.Media{ID: "m1"}, updateErr: errors.New("db down")}
	uc := apppms.NewUpdateMediaUseCase(newMockGormDB(t), mediaRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateMediaInput{ID: "m1", AltText: "New title"})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
