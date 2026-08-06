package pms_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// newMockGormDBN is like newMockGormDB but for use cases that open more
// than one tenant.WithTenant transaction (media upload: one to validate the
// target exists, one to record the row, with the storage write in between).
func newMockGormDBN(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

type stubMediaProductRepo struct {
	product *pmsDomain.Product
	findErr error
}

func (s *stubMediaProductRepo) Create(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubMediaProductRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Product, error) {
	return s.product, s.findErr
}
func (s *stubMediaProductRepo) FindBySlug(_ context.Context, _ string) (*pmsDomain.Product, error) {
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubMediaProductRepo) List(_ context.Context, _ pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	return nil, 0, nil
}
func (s *stubMediaProductRepo) Update(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubMediaProductRepo) Delete(_ context.Context, _ string) error             { return nil }

type stubMediaVariantRepo struct {
	variant *pmsDomain.Variant
	findErr error
}

func (s *stubMediaVariantRepo) Create(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubMediaVariantRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Variant, error) {
	return s.variant, s.findErr
}
func (s *stubMediaVariantRepo) ListByProduct(_ context.Context, _ string) ([]pmsDomain.Variant, error) {
	return nil, nil
}
func (s *stubMediaVariantRepo) Update(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubMediaVariantRepo) Delete(_ context.Context, _ string) error             { return nil }

type stubMediaRepo struct {
	byProduct  []pmsDomain.Media
	attached   []pmsDomain.Media
	attachErr  error
	deleted    []string
	findResult *pmsDomain.Media
	findErr    error
	updated    []pmsDomain.Media
	updateErr  error
}

func (s *stubMediaRepo) Attach(_ context.Context, m *pmsDomain.Media) error {
	if s.attachErr != nil {
		return s.attachErr
	}
	m.ID = "media-1"
	s.attached = append(s.attached, *m)
	return nil
}
func (s *stubMediaRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Media, error) {
	if s.findResult == nil && s.findErr == nil {
		return nil, pmsDomain.ErrMediaNotFound
	}
	return s.findResult, s.findErr
}
func (s *stubMediaRepo) ListByProduct(_ context.Context, _ string) ([]pmsDomain.Media, error) {
	return s.byProduct, nil
}
func (s *stubMediaRepo) Update(_ context.Context, m *pmsDomain.Media) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *m)
	return nil
}
func (s *stubMediaRepo) Delete(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}

type stubFileStorage struct {
	savedTenantID string
	savedFilename string
	savedData     []byte
	saveErr       error
	returnedURL   string
	deletedURL    string
	deleteErr     error
}

func (s *stubFileStorage) Save(_ context.Context, tenantID, filename string, data io.Reader) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	s.savedTenantID = tenantID
	s.savedFilename = filename
	b, _ := io.ReadAll(data)
	s.savedData = b
	if s.returnedURL != "" {
		return s.returnedURL, nil
	}
	return "https://api.example.com/uploads/" + tenantID + "/generated.jpg", nil
}

func (s *stubFileStorage) Delete(_ context.Context, url string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedURL = url
	return nil
}

func (s *stubFileStorage) Get(_ context.Context, url string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.savedData)), nil
}
