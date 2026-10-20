package oms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func cashAndCard() omsDomain.SalesOrderPayments {
	return omsDomain.SalesOrderPayments{
		{Method: "Cash", AmountMinorUnits: 500},
		{Method: "Revolut", AmountMinorUnits: 1500, TransactionID: "rev-1"},
	}
}

func TestSalesOrderPayments_RequireTotal_acceptsPartsThatAddUpToTheTotal(t *testing.T) {
	assert.NoError(t, cashAndCard().RequireTotal(2000))
}

func TestSalesOrderPayments_RequireTotal_rejectsPartsThatDoNotAddUp(t *testing.T) {
	assert.ErrorIs(t, cashAndCard().RequireTotal(1999), omsDomain.ErrPaymentsTotalMismatch)
}

func TestSalesOrderPayments_RequireTotal_acceptsNoPaymentsAtAll(t *testing.T) {
	assert.NoError(t, omsDomain.SalesOrderPayments(nil).RequireTotal(2000))
}

func TestSalesOrderPayments_Method_isTheOnlyPaymentsMethod(t *testing.T) {
	payments := omsDomain.SalesOrderPayments{{Method: "Cash", AmountMinorUnits: 2000}}

	assert.Equal(t, "Cash", payments.Method())
}

func TestSalesOrderPayments_Method_isSplitForSeveralPayments(t *testing.T) {
	assert.Equal(t, omsDomain.SplitPaymentMethod, cashAndCard().Method())
}

func TestSalesOrderPayments_Method_isEmptyWithoutPayments(t *testing.T) {
	assert.Empty(t, omsDomain.SalesOrderPayments(nil).Method())
}

func TestSalesOrderPayments_TransactionID_isTheOnlyPaymentsReference(t *testing.T) {
	payments := omsDomain.SalesOrderPayments{{Method: "Revolut", AmountMinorUnits: 2000, TransactionID: "rev-1"}}

	assert.Equal(t, "rev-1", payments.TransactionID())
}

func TestSalesOrderPayments_TransactionID_isEmptyForSeveralPayments(t *testing.T) {
	assert.Empty(t, cashAndCard().TransactionID())
}
