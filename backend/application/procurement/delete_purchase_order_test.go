package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestDeletePurchaseOrder_deletesExisting(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo, &stubDocumentRepo{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Equal(t, []string{"po-1"}, repo.deleted)
}

func TestDeletePurchaseOrder_propagatesNotFound(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo, &stubDocumentRepo{}, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	require.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

// TestDeletePurchaseOrder_repointsDependentToSurvivingDuplicate is the fix
// for "deleted the flagged PO, no other PO showed as a duplicate anymore" --
// deleting a PO nulls DuplicateOfPurchaseOrderID on anything that pointed
// at it (ON DELETE SET NULL) but IsPotentialDuplicate would otherwise stay
// stuck true with a dangling reference. This PO must get re-pointed at
// whatever other PO still shares its document's content hash.
func TestDeletePurchaseOrder_repointsDependentToSurvivingDuplicate(t *testing.T) {
	repo := &stubPurchaseOrderRepo{
		findResult:   &procurementDomain.PurchaseOrder{ID: "po-1"},
		duplicatesOf: map[string][]procurementDomain.PurchaseOrder{"po-1": {{ID: "po-2"}}},
	}
	documentRepo := &stubDocumentRepo{
		byPO:                []procurementDomain.PurchaseOrderDocument{{ID: "document-2", PurchaseOrderID: "po-2", ContentHash: "hash-x"}},
		duplicateCandidates: []procurementDomain.PurchaseOrderDocument{{ID: "document-3", PurchaseOrderID: "po-3", ContentHash: "hash-x"}},
	}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo, documentRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	require.Len(t, repo.duplicatesSet, 1)
	assert.Equal(t, "po-2", repo.duplicatesSet[0].id)
	assert.Equal(t, "po-3", repo.duplicatesSet[0].duplicateOfPurchaseOrderID)
	assert.Empty(t, repo.cleared)
}

// TestDeletePurchaseOrder_clearsDependentWhenNoOtherDuplicateRemains covers
// the other half: if po-1 was the *only* other PO sharing that content, the
// dependent's flag must clear, not stay stuck true.
func TestDeletePurchaseOrder_clearsDependentWhenNoOtherDuplicateRemains(t *testing.T) {
	repo := &stubPurchaseOrderRepo{
		findResult:   &procurementDomain.PurchaseOrder{ID: "po-1"},
		duplicatesOf: map[string][]procurementDomain.PurchaseOrder{"po-1": {{ID: "po-2"}}},
	}
	documentRepo := &stubDocumentRepo{
		byPO:                []procurementDomain.PurchaseOrderDocument{{ID: "document-2", PurchaseOrderID: "po-2", ContentHash: "hash-x"}},
		duplicateCandidates: nil,
	}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo, documentRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	assert.Empty(t, repo.duplicatesSet)
	assert.Equal(t, []string{"po-2"}, repo.cleared)
}

func TestDeletePurchaseOrder_noDependents_doesNothingExtra(t *testing.T) {
	repo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	documentRepo := &stubDocumentRepo{}
	uc := appprocurement.NewDeletePurchaseOrderUseCase(newMockGormDB(t), repo, documentRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "po-1")

	require.NoError(t, err)
	assert.Empty(t, repo.duplicatesSet)
	assert.Empty(t, repo.cleared)
}
