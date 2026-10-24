package customers

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrCustomerNotFound = errs.New("CUSTOMER_NOT_FOUND", "customer not found")
	// ErrCustomerUnidentified: email is optional, but a customer needs
	// something to be told apart by (see Customer.Identifiable).
	ErrCustomerUnidentified = errs.New("CUSTOMER_UNIDENTIFIED", "a customer needs a name, a company, an email or a phone number")
	// ErrDuplicateWooCustomerID guards Customer.WooCustomerID: at most one
	// customer per tenant may be linked to a given Woo customer
	// (customers_tenant_woo_customer_id_idx). Application code should
	// pre-check via Repository.FindByWooCustomerID (see
	// application/channels.ImportWooCustomersUseCase); this is the
	// database-level backstop for the race between that check and Create.
	ErrDuplicateWooCustomerID = errs.New("CUSTOMER_DUPLICATE_WOO_ID", "woo customer id already linked to another customer")
)
