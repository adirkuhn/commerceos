package iam_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
)

func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

type stubScannerPairingCodeRepo struct {
	findByIDResult    *iamDomain.ScannerPairingCode
	findByIDErr       error
	findPendingResult *iamDomain.ScannerPairingCode
	findPendingErr    error
	createErr         error
	created           []iamDomain.ScannerPairingCode
	markClaimedErr    error
	markClaimedCalls  []string
}

func (s *stubScannerPairingCodeRepo) Create(_ context.Context, p *iamDomain.ScannerPairingCode) error {
	if s.createErr != nil {
		return s.createErr
	}
	p.ID = "pairing-1"
	s.created = append(s.created, *p)
	return nil
}
func (s *stubScannerPairingCodeRepo) FindByID(_ context.Context, _ string) (*iamDomain.ScannerPairingCode, error) {
	return s.findByIDResult, s.findByIDErr
}
func (s *stubScannerPairingCodeRepo) FindPendingByCode(_ context.Context, _ string) (*iamDomain.ScannerPairingCode, error) {
	return s.findPendingResult, s.findPendingErr
}
func (s *stubScannerPairingCodeRepo) MarkClaimed(_ context.Context, id string, _ time.Time) error {
	s.markClaimedCalls = append(s.markClaimedCalls, id)
	return s.markClaimedErr
}
