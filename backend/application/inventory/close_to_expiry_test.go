package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubCloseToExpiryReader struct {
	queries  []invDomain.CloseToExpiryQuery
	items    []invDomain.CloseToExpiryBatch
	counts   map[int]invDomain.CloseToExpiryCounts
	countArg []string
	err      error
}

func (s *stubCloseToExpiryReader) ListCloseToExpiryBatches(_ context.Context, q invDomain.CloseToExpiryQuery) ([]invDomain.CloseToExpiryBatch, error) {
	s.queries = append(s.queries, q)
	return s.items, s.err
}

func (s *stubCloseToExpiryReader) CountCloseToExpiryBatches(_ context.Context, from, to time.Time, windowDays int) (invDomain.CloseToExpiryCounts, error) {
	s.countArg = append(s.countArg, from.Format("2006-01-02")+".."+to.Format("2006-01-02"))
	return s.counts[windowDays], s.err
}

type stubExpiryAckRepo struct {
	created    []invDomain.ExpiryAcknowledgement
	exists     bool
	createErr  error
	deleted    []string
	deleteMiss bool
	deleteErr  error
}

func (s *stubExpiryAckRepo) Create(_ context.Context, ack invDomain.ExpiryAcknowledgement) (bool, error) {
	if s.createErr != nil {
		return false, s.createErr
	}
	s.created = append(s.created, ack)
	return !s.exists, nil
}

func (s *stubExpiryAckRepo) Delete(_ context.Context, batchID string, windowDays int) (bool, error) {
	if s.deleteErr != nil {
		return false, s.deleteErr
	}
	s.deleted = append(s.deleted, batchID)
	return !s.deleteMiss, nil
}

// 2026-10-12 08:00 in Amsterdam.
var expiryNow = func() time.Time { return time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC) }

func amsterdamSettings(windows ...int) *settingsDomain.TenantSettings {
	return &settingsDomain.TenantSettings{Timezone: "Europe/Amsterdam", ExpiryAlert: settingsDomain.ExpiryAlert{Windows: windows}}
}

func tenantCtx() context.Context { return tenant.WithTenantID(context.Background(), "tenant-1") }

func TestListCloseToExpiry_DefaultsToTheLargestWindowWithCountsPerWindow(t *testing.T) {
	reader := &stubCloseToExpiryReader{
		items: []invDomain.CloseToExpiryBatch{{BatchID: "b-1"}},
		counts: map[int]invDomain.CloseToExpiryCounts{
			90: {Open: 3, Acknowledged: 1}, 30: {Open: 2}, 15: {}, 7: {Open: 1},
		},
	}
	uc := appinventory.NewListCloseToExpiryBatchesUseCase(newMockGormDB(t), &stubSettingsRepo{result: amsterdamSettings()}, reader, expiryNow)

	res, err := uc.Execute(tenantCtx(), appinventory.ListCloseToExpiryInput{})

	require.NoError(t, err)
	assert.Equal(t, 90, res.Window.Band.WindowDays)
	assert.Equal(t, "2026-11-12", res.Window.From.Format("2006-01-02")) // 31 days left
	assert.Equal(t, "2027-01-10", res.Window.To.Format("2006-01-02"))   // 90 days left
	require.Len(t, res.Windows, 4)
	assert.Equal(t, []int{90, 30, 15, 7}, []int{res.Windows[0].Band.WindowDays, res.Windows[1].Band.WindowDays, res.Windows[2].Band.WindowDays, res.Windows[3].Band.WindowDays})
	assert.Equal(t, invDomain.CloseToExpiryCounts{Open: 3, Acknowledged: 1}, res.Windows[0].Counts)
	assert.Equal(t, 3, res.Total, "total of the open list")
	assert.Equal(t, []invDomain.CloseToExpiryBatch{{BatchID: "b-1"}}, res.Items)
	assert.Equal(t, "2026-10-12", res.Today.Format("2006-01-02"))
	require.Len(t, reader.queries, 1)
	assert.Equal(t, invDomain.CloseToExpiryQuery{From: res.Window.From, To: res.Window.To, WindowDays: 90, Limit: 50, Offset: 0}, reader.queries[0])
}

func TestListCloseToExpiry_PagesTheAcknowledgedListOfAChosenWindow(t *testing.T) {
	reader := &stubCloseToExpiryReader{counts: map[int]invDomain.CloseToExpiryCounts{30: {Open: 1, Acknowledged: 7}}}
	uc := appinventory.NewListCloseToExpiryBatchesUseCase(newMockGormDB(t), &stubSettingsRepo{result: amsterdamSettings(90, 30)}, reader, expiryNow)

	res, err := uc.Execute(tenantCtx(), appinventory.ListCloseToExpiryInput{WindowDays: 30, Acknowledged: true, Page: 2, PerPage: 5})

	require.NoError(t, err)
	assert.Len(t, res.Windows, 2)
	assert.Equal(t, 0, res.Window.Band.MinDaysLeft)
	assert.Equal(t, 7, res.Total)
	assert.Equal(t, 2, res.Page)
	assert.Equal(t, 5, res.PerPage)
	assert.Equal(t, invDomain.CloseToExpiryQuery{From: res.Window.From, To: res.Window.To, WindowDays: 30, Acknowledged: true, Limit: 5, Offset: 5}, reader.queries[0])
}

func TestListCloseToExpiry_CapsPerPage(t *testing.T) {
	reader := &stubCloseToExpiryReader{}
	uc := appinventory.NewListCloseToExpiryBatchesUseCase(newMockGormDB(t), &stubSettingsRepo{}, reader, expiryNow)

	res, err := uc.Execute(tenantCtx(), appinventory.ListCloseToExpiryInput{PerPage: 1000})

	require.NoError(t, err)
	assert.Equal(t, 200, res.PerPage)
}

func TestListCloseToExpiry_RejectsAWindowNotInUse(t *testing.T) {
	uc := appinventory.NewListCloseToExpiryBatchesUseCase(newMockGormDB(t), &stubSettingsRepo{result: amsterdamSettings(90, 30)}, &stubCloseToExpiryReader{}, expiryNow)

	_, err := uc.Execute(tenantCtx(), appinventory.ListCloseToExpiryInput{WindowDays: 7})

	assert.ErrorIs(t, err, invDomain.ErrUnknownExpiryWindow)
}

func TestListCloseToExpiry_PropagatesReaderErrors(t *testing.T) {
	boom := errors.New("boom")
	uc := appinventory.NewListCloseToExpiryBatchesUseCase(newMockGormDB(t), &stubSettingsRepo{}, &stubCloseToExpiryReader{err: boom}, expiryNow)

	_, err := uc.Execute(tenantCtx(), appinventory.ListCloseToExpiryInput{})

	assert.ErrorIs(t, err, boom)
}

func expiringBatch(daysFromToday int) *invDomain.Batch {
	expiry := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC).AddDate(0, 0, daysFromToday)
	return &invDomain.Batch{ID: "b-1", TenantID: "tenant-1", SKUID: "sku-1", WarehouseID: "wh-1", LotNumber: "L1", ExpiryDate: &expiry}
}

func TestAcknowledgeBatchExpiry_StoresItAndRaisesTheEvent(t *testing.T) {
	acks := &stubExpiryAckRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appinventory.NewAcknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubSettingsRepo{result: amsterdamSettings()},
		&stubBatchRepo{findResult: expiringBatch(20)}, acks, outbox, expiryNow)

	err := uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30, By: "u-1"})

	require.NoError(t, err)
	require.Len(t, acks.created, 1)
	assert.Equal(t, invDomain.ExpiryAcknowledgement{TenantID: "tenant-1", BatchID: "b-1", WindowDays: 30, AcknowledgedBy: "u-1", AcknowledgedAt: expiryNow()}, acks.created[0])
	ev, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "inventory.batch_expiry.acknowledged", ev.Subject)
	assert.Equal(t, map[string]any{"window_days": float64(30), "sku_id": "sku-1", "warehouse_id": "wh-1", "lot_number": "L1"}, ev.Data())
}

func TestAcknowledgeBatchExpiry_AlreadyAcknowledgedRaisesNothing(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appinventory.NewAcknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubSettingsRepo{},
		&stubBatchRepo{findResult: expiringBatch(20)}, &stubExpiryAckRepo{exists: true}, outbox, expiryNow)

	require.NoError(t, uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30, By: "u-1"}))
	assert.Empty(t, outbox.Events)
}

func TestAcknowledgeBatchExpiry_RejectsABatchOutsideTheWindow(t *testing.T) {
	acks := &stubExpiryAckRepo{}
	uc := appinventory.NewAcknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubSettingsRepo{},
		&stubBatchRepo{findResult: expiringBatch(5)}, acks, &eventrecordtest.Outbox{}, expiryNow)

	err := uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30, By: "u-1"})

	assert.ErrorIs(t, err, invDomain.ErrBatchNotInExpiryWindow)
	assert.Empty(t, acks.created)
}

func TestAcknowledgeBatchExpiry_RejectsAWindowNotInUse(t *testing.T) {
	uc := appinventory.NewAcknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubSettingsRepo{result: amsterdamSettings(90)},
		&stubBatchRepo{findResult: expiringBatch(20)}, &stubExpiryAckRepo{}, &eventrecordtest.Outbox{}, expiryNow)

	err := uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30, By: "u-1"})

	assert.ErrorIs(t, err, invDomain.ErrUnknownExpiryWindow)
}

func TestAcknowledgeBatchExpiry_PropagatesBatchNotFound(t *testing.T) {
	uc := appinventory.NewAcknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubSettingsRepo{},
		&stubBatchRepo{findErr: invDomain.ErrBatchNotFound}, &stubExpiryAckRepo{}, &eventrecordtest.Outbox{}, expiryNow)

	err := uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30, By: "u-1"})

	assert.ErrorIs(t, err, invDomain.ErrBatchNotFound)
}

func TestUnacknowledgeBatchExpiry_RemovesItAndRaisesTheEvent(t *testing.T) {
	acks := &stubExpiryAckRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appinventory.NewUnacknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubBatchRepo{findResult: expiringBatch(20)}, acks, outbox)

	err := uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30})

	require.NoError(t, err)
	assert.Equal(t, []string{"b-1"}, acks.deleted)
	ev, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "inventory.batch_expiry.unacknowledged", ev.Subject)
	assert.Equal(t, float64(30), ev.Data()["window_days"])
}

func TestUnacknowledgeBatchExpiry_NothingToRemoveRaisesNothing(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appinventory.NewUnacknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubBatchRepo{findResult: expiringBatch(20)}, &stubExpiryAckRepo{deleteMiss: true}, outbox)

	require.NoError(t, uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30}))
	assert.Empty(t, outbox.Events)
}

func TestUnacknowledgeBatchExpiry_PropagatesBatchNotFound(t *testing.T) {
	uc := appinventory.NewUnacknowledgeBatchExpiryUseCase(newMockGormDB(t), &stubBatchRepo{findErr: invDomain.ErrBatchNotFound}, &stubExpiryAckRepo{}, &eventrecordtest.Outbox{})

	err := uc.Execute(tenantCtx(), appinventory.BatchExpiryAcknowledgementInput{BatchID: "b-1", WindowDays: 30})

	assert.ErrorIs(t, err, invDomain.ErrBatchNotFound)
}
