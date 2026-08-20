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

func TestUpdateCategory_persistsTargetMarginBasisPoints(t *testing.T) {
	categoryRepo := &stubCategoryRepo{}
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), categoryRepo)

	margin := 3500
	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets", TargetMarginBasisPoints: &margin})

	require.NoError(t, err)
	require.NotNil(t, categoryRepo.updated.TargetMarginBasisPoints)
	assert.Equal(t, 3500, *categoryRepo.updated.TargetMarginBasisPoints)
}

func TestUpdateCategory_clearsTargetMarginBasisPointsWhenNil(t *testing.T) {
	categoryRepo := &stubCategoryRepo{findByIDResult: &pmsDomain.Category{ID: "category-1", TargetMarginBasisPoints: intPtr(3000)}}
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), categoryRepo)

	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets"})

	require.NoError(t, err)
	assert.Nil(t, categoryRepo.updated.TargetMarginBasisPoints)
}

func TestUpdateCategory_rejectsOutOfRangeMargin(t *testing.T) {
	uc := apppms.NewUpdateCategoryUseCase(newMockGormDB(t), &stubCategoryRepo{})

	tooHigh := 10000
	_, err := uc.Execute(context.Background(), apppms.UpdateCategoryInput{ID: "category-1", Name: "Widgets", TargetMarginBasisPoints: &tooHigh})

	require.Error(t, err)
	assert.True(t, errors.Is(err, pmsDomain.ErrInvalidTargetMarginBasisPoints))
}

func intPtr(v int) *int { return &v }
