package customers

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrCustomerNotFound = errs.New("CUSTOMER_NOT_FOUND", "customer not found")
	// ErrCustomerUnidentified: email is optional, but a customer needs
	// something to be told apart by (see Customer.Identifiable).
	ErrCustomerUnidentified = errs.New("CUSTOMER_UNIDENTIFIED", "a customer needs a name, a company, an email or a phone number")
	ErrCompanyNameRequired  = errs.New("CUSTOMER_COMPANY_NAME_REQUIRED", "a company customer needs a company name")
	ErrInvalidCustomerType  = errs.New("CUSTOMER_INVALID_TYPE", "a customer is a company or private")
	// ErrDuplicateClientNumber: client numbers are unique per tenant.
	ErrDuplicateClientNumber       = errs.New("CUSTOMER_DUPLICATE_CLIENT_NUMBER", "another customer already has this client number")
	ErrContactPersonNotFound       = errs.New("CONTACT_PERSON_NOT_FOUND", "contact person not found")
	ErrContactPersonNameRequired   = errs.New("CONTACT_PERSON_NAME_REQUIRED", "a contact person needs a first or last name")
	ErrShippingAddressNotFound     = errs.New("CUSTOMER_SHIPPING_ADDRESS_NOT_FOUND", "shipping address not found")
	ErrShippingAddressLineRequired = errs.New("CUSTOMER_SHIPPING_ADDRESS_LINE_REQUIRED", "a shipping address needs an address line")
	ErrShippingLabelTooLong        = errs.New("CUSTOMER_SHIPPING_ADDRESS_LABEL_TOO_LONG", "a shipping address label is at most 60 characters")
	ErrTooManyShippingAddresses    = errs.New("CUSTOMER_SHIPPING_ADDRESS_LIMIT", "a customer has at most 20 shipping addresses")
	// ErrDuplicateWooCustomerID guards Customer.WooCustomerID: at most one
	// customer per tenant may be linked to a given Woo customer
	// (customers_tenant_woo_customer_id_idx). Application code should
	// pre-check via Repository.FindByWooCustomerID (see
	// application/channels.ImportWooCustomersUseCase); this is the
	// database-level backstop for the race between that check and Create.
	ErrDuplicateWooCustomerID = errs.New("CUSTOMER_DUPLICATE_WOO_ID", "woo customer id already linked to another customer")
)
