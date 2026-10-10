package messagecenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/locale"
	mc "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestOrderStatusLabel(t *testing.T) {
	assert.Equal(t, "Shipped", mc.OrderStatusLabel("shipped", locale.English))
	assert.Equal(t, "Verzonden", mc.OrderStatusLabel("shipped", locale.Dutch))
	assert.Equal(t, "Enviado", mc.OrderStatusLabel("shipped", locale.Portuguese))
	assert.Equal(t, "On hold", mc.OrderStatusLabel("on-hold", ""))
	assert.Equal(t, "Something-new", mc.OrderStatusLabel("something-new", locale.Dutch), "unknown status shows itself, capitalized")
	for _, s := range []string{"pending", "processing", "on-hold", "shipped", "completed", "cancelled", "refunded", "failed", "partially_refunded"} {
		for _, l := range []locale.Locale{locale.English, locale.Portuguese, locale.Dutch} {
			assert.NotEqual(t, s, mc.OrderStatusLabel(s, l), "%s in %s", s, l)
		}
	}
}
