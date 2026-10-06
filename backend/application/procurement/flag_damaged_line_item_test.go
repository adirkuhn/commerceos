package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFlagDamagedFixture() (*stubReceivingRecordRepo, *stubLineItemRepo, *stubReceivingDiscrepancyRepo) {
	recRepo := &stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{ID: "receiving-record-1", PurchaseOrderID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{findResult: &procurementDomain.PurchaseOrderLineItem{ID: "line-1", PurchaseOrderID: "po-1", Quantity: 10}}
	discrepancyRepo := &stubReceivingDiscrepancyRepo{}
	return recRepo, lineItemRepo, discrepancyRepo
}

func TestFlagDamagedLineItem_happyPath(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recRepo, lineItemRepo, discrepancyRepo := newFlagDamagedFixture()
	uc := appprocurement.NewFlagDamagedLineItemUseCase(db, recRepo, lineItemRepo, discrepancyRepo, &eventrecordtest.Outbox{})

	d, err := uc.Execute(context.Background(), "receiving-record-1", "line-1", 3, "crushed box")

	require.NoError(t, err)
	assert.Equal(t, procurementDomain.DiscrepancyTypeDamagedItem, d.Type)
	assert.Equal(t, int64(3), d.ReceivedQty)
	assert.Equal(t, "crushed box", d.Details)
	require.NotNil(t, d.LineItemID)
	assert.Equal(t, "line-1", *d.LineItemID)
	require.Len(t, discrepancyRepo.created, 1)
}

func TestFlagDamagedLineItem_invalidQuantity(t *testing.T) {
	recRepo, lineItemRepo, discrepancyRepo := newFlagDamagedFixture()
	uc := appprocurement.NewFlagDamagedLineItemUseCase(nil, recRepo, lineItemRepo, discrepancyRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "receiving-record-1", "line-1", 0, "")

	assert.ErrorIs(t, err, procurementDomain.ErrInvalidDamagedQuantity)
	assert.Empty(t, discrepancyRepo.created)
}

func TestFlagDamagedLineItem_receivingRecordNotFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	_, lineItemRepo, discrepancyRepo := newFlagDamagedFixture()
	recRepo := &stubReceivingRecordRepo{findErr: procurementDomain.ErrReceivingRecordNotFound}
	uc := appprocurement.NewFlagDamagedLineItemUseCase(db, recRepo, lineItemRepo, discrepancyRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "receiving-record-1", "line-1", 3, "")

	assert.ErrorIs(t, err, procurementDomain.ErrReceivingRecordNotFound)
}

func TestFlagDamagedLineItem_lineItemNotFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recRepo, _, discrepancyRepo := newFlagDamagedFixture()
	lineItemRepo := &stubLineItemRepo{findErr: procurementDomain.ErrLineItemNotFound}
	uc := appprocurement.NewFlagDamagedLineItemUseCase(db, recRepo, lineItemRepo, discrepancyRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "receiving-record-1", "line-1", 3, "")

	assert.ErrorIs(t, err, procurementDomain.ErrLineItemNotFound)
}

func TestFlagDamagedLineItem_lineItemBelongsToDifferentPO(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recRepo, lineItemRepo, discrepancyRepo := newFlagDamagedFixture()
	lineItemRepo.findResult.PurchaseOrderID = "po-2"
	uc := appprocurement.NewFlagDamagedLineItemUseCase(db, recRepo, lineItemRepo, discrepancyRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "receiving-record-1", "line-1", 3, "")

	assert.ErrorIs(t, err, procurementDomain.ErrDiscrepancyMismatch)
	assert.Empty(t, discrepancyRepo.created)
}
