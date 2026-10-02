package channels

import (
	appoms "github.com/adirkuhn/commerceos/application/oms"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// applyWooPaymentChanges copies Woo's payment info (method, title,
// transaction id, paid date) onto in wherever it differs from existing,
// reporting whether anything changed. Woo's value only ever fills in or
// replaces -- an empty value on Woo's side never blanks one CommerceOS
// already has (e.g. stamped by its own payment webhook). Shared by the
// order import's re-sync of a known order and BackfillWooOrderPayments.
func applyWooPaymentChanges(existing omsDomain.SalesOrder, wo integrationsDomain.WooOrder, in *appoms.UpdateSalesOrderInput) bool {
	changed := false
	setIfChanged := func(current, incoming string, target **string) {
		if incoming == "" || incoming == current {
			return
		}
		v := incoming
		*target = &v
		changed = true
	}
	setIfChanged(existing.PaymentMethod, wo.PaymentMethod, &in.PaymentMethod)
	setIfChanged(existing.PaymentMethodTitle, wo.PaymentMethodTitle, &in.PaymentMethodTitle)
	setIfChanged(existing.TransactionID, wo.TransactionID, &in.TransactionID)
	if paid, err := parseWooGMTTime(wo.DatePaid); err == nil && (existing.DatePaid == nil || !existing.DatePaid.Equal(paid)) {
		in.DatePaid = &paid
		changed = true
	}
	return changed
}
