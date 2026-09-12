package pms_test

import (
	"context"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubBarcodeAliasRepo struct {
	created          *pmsDomain.BarcodeAlias
	createErr        error
	findByCodeResult *pmsDomain.BarcodeAlias
	findByCodeErr    error
	listBySKUResult  []pmsDomain.BarcodeAlias
	listBySKUErr     error
	deletedID        string
	deleteErr        error
}

func (s *stubBarcodeAliasRepo) Create(_ context.Context, a *pmsDomain.BarcodeAlias) error {
	if s.createErr != nil {
		return s.createErr
	}
	a.ID = "alias-1"
	s.created = a
	return nil
}
func (s *stubBarcodeAliasRepo) FindByCode(_ context.Context, _ string) (*pmsDomain.BarcodeAlias, error) {
	if s.findByCodeResult == nil && s.findByCodeErr == nil {
		return nil, pmsDomain.ErrBarcodeAliasNotFound
	}
	return s.findByCodeResult, s.findByCodeErr
}
func (s *stubBarcodeAliasRepo) ListBySKU(_ context.Context, _ string) ([]pmsDomain.BarcodeAlias, error) {
	return s.listBySKUResult, s.listBySKUErr
}
func (s *stubBarcodeAliasRepo) ListBySKUIDs(_ context.Context, _ []string) ([]pmsDomain.BarcodeAlias, error) {
	return s.listBySKUResult, s.listBySKUErr
}
func (s *stubBarcodeAliasRepo) Delete(_ context.Context, id string) error {
	s.deletedID = id
	return s.deleteErr
}

type stubScaleCodeAliasRepo struct {
	created          *pmsDomain.ScaleCodeAlias
	createErr        error
	findByCodeResult *pmsDomain.ScaleCodeAlias
	findByCodeErr    error
	listBySKUResult  []pmsDomain.ScaleCodeAlias
	listBySKUErr     error
	deletedID        string
	deleteErr        error
}

func (s *stubScaleCodeAliasRepo) Create(_ context.Context, a *pmsDomain.ScaleCodeAlias) error {
	if s.createErr != nil {
		return s.createErr
	}
	a.ID = "scalealias-1"
	s.created = a
	return nil
}
func (s *stubScaleCodeAliasRepo) FindByCode(_ context.Context, _ string) (*pmsDomain.ScaleCodeAlias, error) {
	if s.findByCodeResult == nil && s.findByCodeErr == nil {
		return nil, pmsDomain.ErrScaleCodeAliasNotFound
	}
	return s.findByCodeResult, s.findByCodeErr
}
func (s *stubScaleCodeAliasRepo) ListBySKU(_ context.Context, _ string) ([]pmsDomain.ScaleCodeAlias, error) {
	return s.listBySKUResult, s.listBySKUErr
}
func (s *stubScaleCodeAliasRepo) ListBySKUIDs(_ context.Context, _ []string) ([]pmsDomain.ScaleCodeAlias, error) {
	return s.listBySKUResult, s.listBySKUErr
}
func (s *stubScaleCodeAliasRepo) Delete(_ context.Context, id string) error {
	s.deletedID = id
	return s.deleteErr
}
