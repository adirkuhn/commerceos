package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReceivingRecord_returnsRecordAndDiscrepancies(t *testing.T) {
	db := newMockGormDB(t)
	recRepo := &stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{ID: "receiving-record-1", PurchaseOrderID: "po-1"}}
	discrepancyRepo := &stubReceivingDiscrepancyRepo{byRecord: []procurementDomain.ReceivingDiscrepancy{
		{ID: "discrepancy-1", ReceivingRecordID: "receiving-record-1", Type: procurementDomain.DiscrepancyTypeMissingItem},
	}}
	uc := appprocurement.NewGetReceivingRecordUseCase(db, recRepo, discrepancyRepo)

	detail, err := uc.Execute(context.Background(), "receiving-record-1")

	require.NoError(t, err)
	assert.Equal(t, "receiving-record-1", detail.ReceivingRecord.ID)
	require.Len(t, detail.Discrepancies, 1)
	assert.Equal(t, procurementDomain.DiscrepancyTypeMissingItem, detail.Discrepancies[0].Type)
}

func TestGetReceivingRecord_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	recRepo := &stubReceivingRecordRepo{findErr: procurementDomain.ErrReceivingRecordNotFound}
	uc := appprocurement.NewGetReceivingRecordUseCase(db, recRepo, &stubReceivingDiscrepancyRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrReceivingRecordNotFound)
}
