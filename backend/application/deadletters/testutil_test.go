package deadletters_test

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

var errBoom = errors.New("boom")

// superAdmin is the context the admin API hands a use case for a
// super_admin request.
func superAdmin() context.Context {
	ctx := tenant.WithSuperAdmin(context.Background())
	ctx = tenant.WithUserID(ctx, "u-admin")
	return tenant.WithRole(ctx, tenant.RoleSuperAdmin)
}

// mockDB satisfies n tenant.WithTenant transactions (Begin/set_config/Commit).
func mockDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return db
}

type stubStore struct {
	byKey     map[deadletter.Stream]map[uint64]deadletter.DeadLetter
	page      deadletter.Page
	gotFilter deadletter.Filter
	listErr   error
	getErr    error
	deleteErr error
	deleted   []uint64
}

func storeWith(dls ...deadletter.DeadLetter) *stubStore {
	s := &stubStore{byKey: map[deadletter.Stream]map[uint64]deadletter.DeadLetter{}}
	for _, d := range dls {
		if s.byKey[d.Stream] == nil {
			s.byKey[d.Stream] = map[uint64]deadletter.DeadLetter{}
		}
		s.byKey[d.Stream][d.Sequence] = d
	}
	return s
}

func (s *stubStore) List(_ context.Context, f deadletter.Filter) (deadletter.Page, error) {
	s.gotFilter = f
	return s.page, s.listErr
}

func (s *stubStore) Get(_ context.Context, stream deadletter.Stream, seq uint64) (deadletter.DeadLetter, error) {
	if s.getErr != nil {
		return deadletter.DeadLetter{}, s.getErr
	}
	d, ok := s.byKey[stream][seq]
	if !ok {
		return deadletter.DeadLetter{}, deadletter.ErrNotFound
	}
	return d, nil
}

func (s *stubStore) Delete(_ context.Context, _ deadletter.Stream, seq uint64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, seq)
	return nil
}

type stubPublisher struct {
	err       error
	published []deadletter.DeadLetter
}

func (p *stubPublisher) Republish(_ context.Context, d deadletter.DeadLetter) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, d)
	return nil
}

func sampleDeadLetter() deadletter.DeadLetter {
	return deadletter.DeadLetter{
		Stream: deadletter.StreamEvents, Sequence: 7, Subject: "oms.sales_order.status_changed",
		Consumer: "sales-order-status-changed-scheduler", EventID: "e-1", TenantID: "t-1",
		Error: "woo: 403", NumDelivered: 5, Data: []byte(`{"event_id":"e-1","tenant_id":"t-1"}`),
	}
}
