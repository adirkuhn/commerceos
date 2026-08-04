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

func TestRecordStockMovement_rejectsZeroQuantity(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 0, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
	require.Empty(t, movementRepo.created)
}

func TestRecordStockMovement_rejectsNegativeReceipt(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: -5, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
}

func TestRecordStockMovement_rejectsPositiveSale(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, QuantityDelta: 5, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
}

func TestRecordStockMovement_raisesStockChangedEvent(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	outboxRepo := &stubOutboxRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 10, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, []string{"inventory.stock.changed"}, outboxRepo.calls)
}

func TestRecordStockMovement_acceptsPositiveReceipt(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 10, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, int64(10), m.QuantityDelta)
	require.Len(t, movementRepo.created, 1)
}

func TestRecordStockMovement_acceptsNegativeSale(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeSale, QuantityDelta: -3, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, int64(-3), m.QuantityDelta)
}

func TestRecordStockMovement_adjustmentAllowsNegative(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeAdjustment, QuantityDelta: -3, OccurredAt: time.Now(),
	})
	require.NoError(t, err)
}

func TestRecordStockMovement_adjustmentAllowsPositive(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeAdjustment, QuantityDelta: 3, OccurredAt: time.Now(),
	})
	require.NoError(t, err)
}

func TestRecordStockMovement_acceptsNegativeDamaged(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeDamaged, QuantityDelta: -2, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, int64(-2), m.QuantityDelta)
}

func TestRecordStockMovement_rejectsPositiveDamaged(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeDamaged, QuantityDelta: 2, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
}

func TestRecordStockMovement_acceptsNegativeLost(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	m, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeLost, QuantityDelta: -1, OccurredAt: time.Now(),
	})

	require.NoError(t, err)
	require.Equal(t, int64(-1), m.QuantityDelta)
}

func TestRecordStockMovement_rejectsPositiveLost(t *testing.T) {
	movementRepo := &stubStockMovementRepo{}
	uc := appinv.NewRecordStockMovementUseCase(newMockGormDB(t), movementRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), appinv.RecordStockMovementInput{
		SKUID: "sku-1", WarehouseID: "wh-1", Type: invDomain.MovementTypeLost, QuantityDelta: 1, OccurredAt: time.Now(),
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, invDomain.ErrInvalidQuantity))
}
