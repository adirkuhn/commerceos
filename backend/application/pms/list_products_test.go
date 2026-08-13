package pms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListProducts_defaultsPageAndPerPage(t *testing.T) {
	repo := &stubProductRepo{listResult: []pmsDomain.Product{{ID: "p1"}}, listTotal: 1}
	uc := apppms.NewListProductsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Equal(t, 20, repo.gotFilter.Limit)
	require.Equal(t, 0, repo.gotFilter.Offset)
	require.Nil(t, repo.gotFilter.Search)
}

func TestListProducts_computesOffsetFromPage(t *testing.T) {
	repo := &stubProductRepo{}
	uc := apppms.NewListProductsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{Page: 3, PerPage: 20})

	require.NoError(t, err)
	require.Equal(t, 20, repo.gotFilter.Limit)
	require.Equal(t, 40, repo.gotFilter.Offset)
}

func TestListProducts_capsPerPageAtMax(t *testing.T) {
	repo := &stubProductRepo{}
	uc := apppms.NewListProductsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{PerPage: 10000})

	require.NoError(t, err)
	require.Equal(t, 200, repo.gotFilter.Limit)
}

func TestListProducts_passesSearchThrough(t *testing.T) {
	repo := &stubProductRepo{}
	uc := apppms.NewListProductsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{Search: "shirt"})

	require.NoError(t, err)
	require.NotNil(t, repo.gotFilter.Search)
	require.Equal(t, "shirt", *repo.gotFilter.Search)
}

func TestListProducts_passesUpdatedAfterThrough(t *testing.T) {
	repo := &stubProductRepo{}
	uc := apppms.NewListProductsUseCase(newMockGormDB(t), repo)
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{UpdatedAfter: &since})

	require.NoError(t, err)
	require.NotNil(t, repo.gotFilter.UpdatedAfter)
	require.True(t, since.Equal(*repo.gotFilter.UpdatedAfter))
}

func TestListProducts_propagatesRepoError(t *testing.T) {
	repo := &stubProductRepo{listErr: errors.New("db down")}
	uc := apppms.NewListProductsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), apppms.ListProductsInput{})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
