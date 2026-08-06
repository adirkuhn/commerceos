package procurement_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestListReceivingRecords_returnsRepoResultForPO(t *testing.T) {
	repo := &stubReceivingRecordRepo{byPO: []procurementDomain.ReceivingRecord{{ID: "receiving-record-1", PurchaseOrderID: "po-1"}}}
	uc := appprocurement.NewListReceivingRecordsUseCase(newMockGormDB(t), repo)

	records, err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, "receiving-record-1", records[0].ID)
}
