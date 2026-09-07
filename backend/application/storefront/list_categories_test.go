package storefront_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestListCategories_returnsTheTenantsWholeTree(t *testing.T) {
	parentID := "cat-1"
	categoryRepo := &fakeCategoryRepo{out: []pmsDomain.Category{
		{ID: "cat-1", Name: "Shoes", Slug: "shoes"},
		{ID: "cat-2", Name: "Sneakers", Slug: "sneakers", ParentID: &parentID},
	}}
	uc := appstorefront.NewListCategoriesUseCase(newMockGormDB(t, 1), categoryRepo)

	categories, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, categories, 2)
	require.Nil(t, categories[0].ParentID)
	require.NotNil(t, categories[1].ParentID)
	require.Equal(t, "cat-1", *categories[1].ParentID)
}

func TestListCategories_propagatesRepoError(t *testing.T) {
	uc := appstorefront.NewListCategoriesUseCase(newMockGormDB(t, 1), &fakeCategoryRepo{err: errors.New("db down")})

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
