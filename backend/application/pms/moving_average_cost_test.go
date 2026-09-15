package pms_test

import (
	"testing"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyReceiptToMovingAverageCost_blendsExistingAndReceivedByValue(t *testing.T) {
	existingAvg := int64(440) // 100 * 4.40
	// 100 @ 4.40 existing, receive 50 @ 5.20 -> (440*100 + 520*50) / 150 = 700/1.5... in minor units:
	// existing value 44000, received value 50*520=26000, total 70000 / 150 = 466.67 -> rounds to 467
	got, err := apppms.ApplyReceiptToMovingAverageCost(100, &existingAvg, 50, 520)

	require.NoError(t, err)
	assert.Equal(t, int64(467), got)
}

func TestApplyReceiptToMovingAverageCost_noExistingStockUsesReceivedCostDirectly(t *testing.T) {
	got, err := apppms.ApplyReceiptToMovingAverageCost(0, nil, 100, 440)

	require.NoError(t, err)
	assert.Equal(t, int64(440), got)
}

func TestApplyReceiptToMovingAverageCost_unknownExistingCostMergesAsReceivedCost(t *testing.T) {
	// existing qty present but cost unknown (SKUCostPriceSourceUnknown) --
	// must not treat unknown as 0 (that would understate the blended cost),
	// so the whole lot merges in at the new receipt's cost.
	got, err := apppms.ApplyReceiptToMovingAverageCost(100, nil, 50, 520)

	require.NoError(t, err)
	assert.Equal(t, int64(520), got)
}

func TestApplyReceiptToMovingAverageCost_roundsHalfUp(t *testing.T) {
	existingAvg := int64(1)
	// 1*1 + 1*2 = 3 / 2 = 1.5 -> rounds to 2
	got, err := apppms.ApplyReceiptToMovingAverageCost(1, &existingAvg, 1, 2)

	require.NoError(t, err)
	assert.Equal(t, int64(2), got)
}

func TestApplyReceiptToMovingAverageCost_rejectsNonPositiveReceivedQuantity(t *testing.T) {
	_, err := apppms.ApplyReceiptToMovingAverageCost(100, nil, 0, 440)

	require.Error(t, err)
}

func TestApplyReceiptToMovingAverageCost_rejectsNegativeExistingQuantity(t *testing.T) {
	_, err := apppms.ApplyReceiptToMovingAverageCost(-1, nil, 50, 440)

	require.Error(t, err)
}
