package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListProductMedia_returnsRepoResult(t *testing.T) {
	mediaRepo := &stubMediaRepo{byProduct: []pmsDomain.Media{{ID: "media-1", URL: "https://example.com/x.jpg"}}}
	uc := apppms.NewListProductMediaUseCase(newMockGormDBN(t, 1), mediaRepo)

	out, err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "https://example.com/x.jpg", out[0].URL)
}
