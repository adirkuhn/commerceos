package pms_test

import (
	"context"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubCategoryRepo struct {
	findByIDResult *pmsDomain.Category
	findByIDErr    error
	updated        *pmsDomain.Category
	created        *pmsDomain.Category
}

func (s *stubCategoryRepo) Create(_ context.Context, c *pmsDomain.Category) error {
	c.ID = "category-1"
	s.created = c
	return nil
}
func (s *stubCategoryRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Category, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return &pmsDomain.Category{ID: "category-1"}, nil
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubCategoryRepo) FindBySlug(_ context.Context, _ string) (*pmsDomain.Category, error) {
	return nil, pmsDomain.ErrCategoryNotFound
}
func (s *stubCategoryRepo) List(_ context.Context) ([]pmsDomain.Category, error) { return nil, nil }
func (s *stubCategoryRepo) Update(_ context.Context, c *pmsDomain.Category) error {
	s.updated = c
	return nil
}
func (s *stubCategoryRepo) Delete(_ context.Context, _ string) error { return nil }
