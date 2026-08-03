package notifications_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

// newMockGormDB mirrors application/inventory's helper of the same name.
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

type stubNotificationRepo struct {
	created           []notificationsDomain.Notification
	createErr         error
	listResult        []notificationsDomain.Notification
	listErr           error
	markReadID        string
	markReadErr       error
	markAllReadCalled bool
	markAllReadErr    error
}

func (s *stubNotificationRepo) Create(_ context.Context, n *notificationsDomain.Notification) error {
	if s.createErr != nil {
		return s.createErr
	}
	n.ID = "notif-1"
	s.created = append(s.created, *n)
	return nil
}

func (s *stubNotificationRepo) List(_ context.Context, _ int) ([]notificationsDomain.Notification, error) {
	return s.listResult, s.listErr
}

func (s *stubNotificationRepo) MarkRead(_ context.Context, id string) error {
	s.markReadID = id
	return s.markReadErr
}

func (s *stubNotificationRepo) MarkAllRead(_ context.Context) error {
	s.markAllReadCalled = true
	return s.markAllReadErr
}
