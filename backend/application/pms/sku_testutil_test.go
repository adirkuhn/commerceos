package pms_test

import (
	"context"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubSKURepo struct {
	findByCodeResult *pmsDomain.SKU
	findByCodeErr    error
	findByIDResult   *pmsDomain.SKU
	findByIDErr      error
	created          *pmsDomain.SKU
	createErr        error
	updated          *pmsDomain.SKU
}

func (s *stubSKURepo) Create(_ context.Context, sku *pmsDomain.SKU) error {
	if s.createErr != nil {
		return s.createErr
	}
	sku.ID = "sku-1"
	s.created = sku
	return nil
}
func (s *stubSKURepo) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return &pmsDomain.SKU{ID: "sku-1"}, nil
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubSKURepo) FindByCode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	if s.findByCodeResult == nil && s.findByCodeErr == nil {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return s.findByCodeResult, s.findByCodeErr
}
func (s *stubSKURepo) ListByVariant(_ context.Context, _ string) ([]pmsDomain.SKU, error) {
	return nil, nil
}
func (s *stubSKURepo) Update(_ context.Context, sku *pmsDomain.SKU) error {
	s.updated = sku
	return nil
}
func (s *stubSKURepo) Delete(_ context.Context, _ string) error { return nil }
