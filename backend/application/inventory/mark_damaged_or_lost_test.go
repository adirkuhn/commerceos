package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func TestMarkDamagedOrLost_recordsNegativeDamagedMovement(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", QuantityDelta: 10},
	}}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, &stubOutboxRepo{})

	m, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 3, Type: invDomain.MovementTypeDamaged,
		Reason: "dropped pallet", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, int64(-3), m.QuantityDelta)
	require.Equal(t, invDomain.MovementTypeDamaged, m.Type)
	require.Len(t, movementRepo.created, 1)
}

func TestMarkDamagedOrLost_recordsNegativeLostMovement(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", QuantityDelta: 10},
	}}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, &stubOutboxRepo{})

	m, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 1, Type: invDomain.MovementTypeLost,
		Reason: "count mismatch", OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, int64(-1), m.QuantityDelta)
	require.Equal(t, invDomain.MovementTypeLost, m.Type)
}

func TestMarkDamagedOrLost_rejectsNonPositiveQuantity(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, &stubOutboxRepo{})

	_, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 0, Type: invDomain.MovementTypeDamaged, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
	require.Empty(t, movementRepo.created)
}

func TestMarkDamagedOrLost_rejectsWhenScopeLacksEnoughStock(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", QuantityDelta: 2},
	}}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, &stubOutboxRepo{})

	_, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 3, Type: invDomain.MovementTypeDamaged, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInsufficientStock))
	require.Empty(t, movementRepo.created)
}

func TestMarkDamagedOrLost_rejectsOtherMovementTypes(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, &stubOutboxRepo{})

	_, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 1, Type: invDomain.MovementTypeAdjustment, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidMovementType))
	require.Empty(t, movementRepo.created)
}

func TestMarkDamagedOrLost_setsBatchIDWhenGiven(t *testing.T) {
	batchID := "batch-1"
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID, QuantityDelta: 10},
	}}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, &stubOutboxRepo{})

	m, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID, Quantity: 2, Type: invDomain.MovementTypeDamaged, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.NotNil(t, m.BatchID)
	require.Equal(t, "batch-1", *m.BatchID)
}

func TestMarkDamagedOrLost_raisesStockChangedEvent(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", QuantityDelta: 10},
	}}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, &stubDiscardRecordRepo{}, outboxRepo)

	_, _, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 3, Type: invDomain.MovementTypeDamaged, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

// The whole point of this session's fix: every call now ALSO writes a
// DiscardRecord (see MarkDamagedOrLostUseCase's own doc comment) --
// previously only DiscardBatchUseCase's expiry/scanner flow did, so the
// inventory page's Damaged/Lost button and a booking's write-off replay
// were invisible in the tenant-wide discarded/damaged/lost report.
func TestMarkDamagedOrLost_alsoWritesADiscardRecordLinkedToTheMovement(t *testing.T) {
	movementRepo := &stubStockMovementRepo{bySKU: []invDomain.StockMovement{
		{SKUID: "sku-1", WarehouseID: "wh-1", QuantityDelta: 10},
	}}
	discardRepo := &stubDiscardRecordRepo{}
	uc := appinv.NewMarkDamagedOrLostUseCase(newMockGormDB(t), movementRepo, discardRepo, &stubOutboxRepo{})
	photoURL := "https://example.com/photo.jpg"

	movement, record, err := uc.Execute(context.Background(), appinv.MarkDamagedOrLostInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Quantity: 3, Type: invDomain.MovementTypeDamaged,
		Reason: "dropped pallet", OccurredAt: time.Now(), DiscardedBy: "user-1", PhotoURL: &photoURL,
	})

	require.NoError(t, err)
	require.Len(t, discardRepo.created, 1)
	created := discardRepo.created[0]
	require.Equal(t, movement.ID, created.MovementID)
	require.Equal(t, invDomain.MovementTypeDamaged, created.Type)
	require.Equal(t, int64(3), created.Quantity)
	require.Equal(t, "dropped pallet", created.Reason)
	require.Equal(t, "user-1", created.DiscardedBy)
	require.NotNil(t, created.PhotoURL)
	require.Equal(t, photoURL, *created.PhotoURL)
	require.Equal(t, created, *record)
}
