package oms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// The event is part of the write: an outbox failure fails the use case.
func TestOMSUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()

	inProgressPicking := func() *stubPickingSessionRepo {
		return &stubPickingSessionRepo{findByIDResult: &omsDomain.PickingSession{ID: "s-1", SalesOrderID: "order-1", Status: omsDomain.PickingSessionStatusInProgress}}
	}
	_, err := appoms.NewPausePickingUseCase(newMockGormDB(t), inProgressPicking(), failing).Execute(ctx, "s-1")
	assert.ErrorIs(t, err, boom, "pause picking")

	_, err = appoms.NewResetPickingSessionUseCase(newMockGormDB(t), inProgressPicking(), &stubPickScanRepo{}, failing).Execute(ctx, "s-1")
	assert.ErrorIs(t, err, boom, "reset picking")

	activity := appoms.NewRecordSalesOrderActivityUseCase(newMockGormDB(t),
		&stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1"}}, &stubSalesOrderActivityRepo{}, failing)
	_, err = activity.Execute(ctx, appoms.RecordSalesOrderActivityInput{SalesOrderID: "order-1", ActivityType: "note"})
	assert.ErrorIs(t, err, boom, "activity")

	_, err = appoms.NewStartOrResumePickingUseCase(newMockGormDB(t),
		&stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing}}, &stubOrigins{},
		&stubPickingSessionRepo{}, &stubPackingSessionRepo{}, &stubPackScanRepo{}, failing).
		Execute(ctx, "order-1", omsDomain.LockActor{UserID: "u"})
	assert.ErrorIs(t, err, boom, "start picking")
}
