package promotions_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

func TestClearanceBatch_IsFinished(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	tomorrow := now.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	cases := []struct {
		name  string
		batch promotionsDomain.ClearanceBatch
		want  bool
	}{
		{"has stock, expires later", promotionsDomain.ClearanceBatch{Found: true, Quantity: 5, ExpiryDate: &tomorrow}, false},
		{"has stock, no expiry date", promotionsDomain.ClearanceBatch{Found: true, Quantity: 5}, false},
		{"removed", promotionsDomain.ClearanceBatch{Found: false, Quantity: 5, ExpiryDate: &tomorrow}, true},
		{"sold out", promotionsDomain.ClearanceBatch{Found: true, Quantity: 0, ExpiryDate: &tomorrow}, true},
		{"negative stock", promotionsDomain.ClearanceBatch{Found: true, Quantity: -1}, true},
		{"expiry date passed", promotionsDomain.ClearanceBatch{Found: true, Quantity: 5, ExpiryDate: &yesterday}, true},
		{"expiry date is now", promotionsDomain.ClearanceBatch{Found: true, Quantity: 5, ExpiryDate: &now}, true},
		{"marked expired", promotionsDomain.ClearanceBatch{Found: true, Quantity: 5, ExpiryDate: &tomorrow, MarkedExpired: true}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.batch.IsFinished(now))
		})
	}
}
