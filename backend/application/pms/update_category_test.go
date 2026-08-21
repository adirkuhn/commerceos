package pms_test

import (
	"context"
	"errors"
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCategory_persistsTargetMarkupBasisPoints(t *testing.T) {
	categoryRepo := &stubCategoryRepo{}
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), categoryRepo)

	markup := 3500
	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets", TargetMarkupBasisPoints: &markup})

	require.NoError(t, err)
	require.NotNil(t, categoryRepo.updated.TargetMarkupBasisPoints)
	assert.Equal(t, 3500, *categoryRepo.updated.TargetMarkupBasisPoints)
}

func TestUpdateCategory_clearsTargetMarkupBasisPointsWhenNil(t *testing.T) {
	categoryRepo := &stubCategoryRepo{findByIDResult: &pmsDomain.Category{ID: "category-1", TargetMarkupBasisPoints: intPtr(3000)}}
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), categoryRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets"})

	require.NoError(t, err)
	assert.Nil(t, categoryRepo.updated.TargetMarkupBasisPoints)
}

func TestUpdateCategory_rejectsNegativeMarkup(t *testing.T) {
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), &stubCategoryRepo{})

	negative := -1
	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets", TargetMarkupBasisPoints: &negative})

	require.Error(t, err)
	assert.True(t, errors.Is(err, pmsDomain.ErrInvalidTargetMarkupBasisPoints))
}

func TestUpdateCategory_allowsMarkupOf100PercentAndAbove(t *testing.T) {
	// Unlike margin (capped under 100%), markup has no upper bound.
	categoryRepo := &stubCategoryRepo{}
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), categoryRepo)

	high := 30000
	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets", TargetMarkupBasisPoints: &high})

	require.NoError(t, err)
	require.NotNil(t, categoryRepo.updated.TargetMarkupBasisPoints)
	assert.Equal(t, 30000, *categoryRepo.updated.TargetMarkupBasisPoints)
}

func intPtr(v int) *int { return &v }
