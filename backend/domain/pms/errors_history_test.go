package pms_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// Every "blocked by history" delete error is one kind of failure to the
// API (409 PRODUCT_HAS_HISTORY) while keeping its own message.
func TestProductHistoryErrorsAreErrProductHasHistory(t *testing.T) {
	for _, err := range []error{
		pmsDomain.ErrProductHasReceivingHistory,
		pmsDomain.ErrProductHasSalesOrderHistory,
		pmsDomain.ErrProductHasPickingHistory,
		pmsDomain.ErrProductHasDiscardHistory,
	} {
		assert.True(t, errors.Is(err, pmsDomain.ErrProductHasHistory), err.Error())
	}
}
