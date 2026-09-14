package messagecenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestCategory_Valid(t *testing.T) {
	valid := []messagecenter.Category{
		messagecenter.CategoryPurchase, messagecenter.CategoryOrderStatus,
		messagecenter.CategoryStockInfo, messagecenter.CategoryTracking, messagecenter.CategoryCustom,
	}
	for _, c := range valid {
		assert.True(t, c.Valid(), "expected %q to be valid", c)
	}
	assert.False(t, messagecenter.Category("bogus").Valid())
	assert.False(t, messagecenter.Category("").Valid())
}
