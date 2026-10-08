package oms_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type fakeDiscrepancyRepo struct {
	existing []omsDomain.LineDiscrepancy
	byID     map[string]omsDomain.LineDiscrepancy
	open     []omsDomain.OpenLineDiscrepancy
	created  []omsDomain.LineDiscrepancy
	saved    []omsDomain.LineDiscrepancy
	err      error
}

func (f *fakeDiscrepancyRepo) ListBySalesOrder(context.Context, string) ([]omsDomain.LineDiscrepancy, error) {
	return f.existing, f.err
}
func (f *fakeDiscrepancyRepo) FindByID(_ context.Context, id string) (*omsDomain.LineDiscrepancy, error) {
	d, ok := f.byID[id]
	if !ok {
		return nil, omsDomain.ErrLineDiscrepancyNotFound
	}
	return &d, nil
}
func (f *fakeDiscrepancyRepo) Create(_ context.Context, d *omsDomain.LineDiscrepancy) error {
	if f.err != nil {
		return f.err
	}
	d.ID = "d-new"
	f.created = append(f.created, *d)
	return nil
}
func (f *fakeDiscrepancyRepo) Save(_ context.Context, d *omsDomain.LineDiscrepancy) error {
	f.saved = append(f.saved, *d)
	return f.err
}
func (f *fakeDiscrepancyRepo) ListOpen(context.Context, int, int) ([]omsDomain.OpenLineDiscrepancy, int64, error) {
	return f.open, int64(len(f.open)), f.err
}

var differencesNow = time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)

func newRecordDifferences(t *testing.T, lines []omsDomain.SalesOrderLineItem, repo *fakeDiscrepancyRepo, outbox *eventrecordtest.Outbox) *appoms.RecordChannelOrderDifferencesUseCase {
	t.Helper()
	return appoms.NewRecordChannelOrderDifferencesUseCase(newMockGormDB(t),
		&stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1", TenantID: "tenant-1"}},
		&stubSalesOrderLineItemRepo{listBySalesOrder: lines}, repo, outbox, func() time.Time { return differencesNow })
}

func pinducaInput() appoms.RecordChannelOrderDifferencesInput {
	return appoms.RecordChannelOrderDifferencesInput{
		SalesOrderID: "so-1",
		ChannelLines: []omsDomain.ChannelLine{
			{SKUCode: "7891095031139", Name: "Batata palha", Quantity: 2},
			{SKUCode: "7896015976606", Name: "Mistura de Pão de Queijo - Pinduca - 500g", Quantity: 1},
		},
		ChannelTotalMinorUnits: 3641, ChannelLinesTotalMinorUnits: 3641,
	}
}

func TestRecordChannelOrderDifferences_missingItem_isReportedNotAdded(t *testing.T) {
	repo := &fakeDiscrepancyRepo{}
	outbox := &eventrecordtest.Outbox{}
	lines := []omsDomain.SalesOrderLineItem{{SKUCode: "7891095031139", Name: "Batata palha", Quantity: 2}}
	lineRepo := &stubSalesOrderLineItemRepo{listBySalesOrder: lines}
	uc := appoms.NewRecordChannelOrderDifferencesUseCase(newMockGormDB(t),
		&stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1", TenantID: "tenant-1"}},
		lineRepo, repo, outbox, func() time.Time { return differencesNow })

	require.NoError(t, uc.Execute(context.Background(), pinducaInput()))

	assert.Empty(t, lineRepo.created, "the order itself is never changed")
	require.Len(t, repo.created, 1)
	d := repo.created[0]
	assert.Equal(t, omsDomain.LineDiscrepancyMissing, d.Kind)
	assert.Equal(t, "tenant-1", d.TenantID)
	assert.Equal(t, "so-1", d.SalesOrderID)
	assert.Equal(t, "7896015976606", d.SKUCode)
	assert.Equal(t, differencesNow, d.DetectedAt)

	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "oms.line_discrepancy.detected", event.Subject)
	assert.Equal(t, "missing", event.Data()["kind"])
	assert.Equal(t, "so-1", event.Data()["sales_order_id"])
}

func TestRecordChannelOrderDifferences_channelLinesDontAddUp_reportsTotal(t *testing.T) {
	repo := &fakeDiscrepancyRepo{}
	in := pinducaInput()
	in.ChannelLinesTotalMinorUnits = 3294
	lines := []omsDomain.SalesOrderLineItem{
		{SKUCode: "7891095031139", Quantity: 2}, {SKUCode: "7896015976606", Quantity: 1},
	}

	require.NoError(t, newRecordDifferences(t, lines, repo, &eventrecordtest.Outbox{}).Execute(context.Background(), in))

	require.Len(t, repo.created, 1)
	assert.Equal(t, omsDomain.LineDiscrepancyTotal, repo.created[0].Kind)
	assert.Equal(t, int64(3641), repo.created[0].ChannelTotalMinorUnits)
	assert.Equal(t, int64(3294), repo.created[0].LocalTotalMinorUnits)
}

func TestRecordChannelOrderDifferences_roundingCents_areNotATotalDifference(t *testing.T) {
	repo := &fakeDiscrepancyRepo{}
	in := pinducaInput()
	in.ChannelLinesTotalMinorUnits = 3642
	lines := []omsDomain.SalesOrderLineItem{
		{SKUCode: "7891095031139", Quantity: 2}, {SKUCode: "7896015976606", Quantity: 1},
	}

	require.NoError(t, newRecordDifferences(t, lines, repo, &eventrecordtest.Outbox{}).Execute(context.Background(), in))

	assert.Empty(t, repo.created)
}

func TestRecordChannelOrderDifferences_fixedByHand_resolvesTheDifference(t *testing.T) {
	repo := &fakeDiscrepancyRepo{existing: []omsDomain.LineDiscrepancy{{
		ID: "d-1", SalesOrderID: "so-1", Kind: omsDomain.LineDiscrepancyMissing, ItemKey: "sku:7896015976606",
		SKUCode: "7896015976606", ChannelQuantity: 1, DetectedAt: differencesNow.Add(-time.Hour),
	}}}
	outbox := &eventrecordtest.Outbox{}
	lines := []omsDomain.SalesOrderLineItem{
		{SKUCode: "7891095031139", Quantity: 2}, {SKUCode: "7896015976606", Quantity: 1},
	}

	require.NoError(t, newRecordDifferences(t, lines, repo, outbox).Execute(context.Background(), pinducaInput()))

	require.Len(t, repo.saved, 1)
	require.NotNil(t, repo.saved[0].ResolvedAt)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "oms.line_discrepancy.resolved", event.Subject)
	assert.Equal(t, "d-1", event.Payload["entity_id"])
}

func TestRecordChannelOrderDifferences_nothingDifferent_recordsNothing(t *testing.T) {
	repo := &fakeDiscrepancyRepo{}
	outbox := &eventrecordtest.Outbox{}
	lines := []omsDomain.SalesOrderLineItem{
		{SKUCode: "7891095031139", Quantity: 2}, {SKUCode: "7896015976606", Quantity: 1},
	}

	require.NoError(t, newRecordDifferences(t, lines, repo, outbox).Execute(context.Background(), pinducaInput()))

	assert.Empty(t, repo.created)
	assert.Empty(t, outbox.Events)
}

func TestRecordChannelOrderDifferences_storeFails_returnsError(t *testing.T) {
	repo := &fakeDiscrepancyRepo{err: errors.New("db down")}

	assert.Error(t, newRecordDifferences(t, nil, repo, &eventrecordtest.Outbox{}).Execute(context.Background(), pinducaInput()))
}

func TestDismissLineDiscrepancy_dismissesAndRecordsEvent(t *testing.T) {
	repo := &fakeDiscrepancyRepo{byID: map[string]omsDomain.LineDiscrepancy{
		"d-1": {ID: "d-1", SalesOrderID: "so-1", Kind: omsDomain.LineDiscrepancyMissing, SKUCode: "A"},
	}}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewDismissLineDiscrepancyUseCase(newMockGormDB(t), repo, outbox, func() time.Time { return differencesNow })

	require.NoError(t, uc.Execute(context.Background(), "d-1"))

	require.Len(t, repo.saved, 1)
	require.NotNil(t, repo.saved[0].DismissedAt)
	event, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "oms.line_discrepancy.dismissed", event.Subject)
}

func TestDismissLineDiscrepancy_alreadyDismissed_isRejected(t *testing.T) {
	repo := &fakeDiscrepancyRepo{byID: map[string]omsDomain.LineDiscrepancy{
		"d-1": {ID: "d-1", DismissedAt: &differencesNow},
	}}
	uc := appoms.NewDismissLineDiscrepancyUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{}, time.Now)

	assert.ErrorIs(t, uc.Execute(context.Background(), "d-1"), omsDomain.ErrLineDiscrepancyNotOpen)
}

func TestDismissLineDiscrepancy_unknown_isNotFound(t *testing.T) {
	uc := appoms.NewDismissLineDiscrepancyUseCase(newMockGormDB(t), &fakeDiscrepancyRepo{}, &eventrecordtest.Outbox{}, time.Now)

	assert.ErrorIs(t, uc.Execute(context.Background(), "nope"), omsDomain.ErrLineDiscrepancyNotFound)
}

func TestListOpenLineDiscrepancies_returnsPage(t *testing.T) {
	repo := &fakeDiscrepancyRepo{open: []omsDomain.OpenLineDiscrepancy{{LineDiscrepancy: omsDomain.LineDiscrepancy{ID: "d-1"}}}}
	uc := appoms.NewListOpenLineDiscrepanciesUseCase(newMockGormDB(t), repo)

	got, total, err := uc.Execute(context.Background(), 1, 20)

	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "d-1", got[0].ID)
}
