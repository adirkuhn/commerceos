package events

// Customer actions (backend/application/customers). Personal data
// (name, email, phone, username, addresses, avatar, note) never travels in
// these events: changed personal fields are listed by name in
// data.changed_fields, without values -- read the customer by entity_id.
var (
	CustomerCreated = defineAction("customers.customer.created",
		"A customer was created. data: woo_customer_id, role, customer_type, client_number.")
	CustomerUpdated = defineAction("customers.customer.updated",
		"A customer changed (changes: woo_customer_id, role, ship_to_billing, customer_type, company_name, client_number, kvk_number, vat_number). data.changed_fields: personal fields that changed (email, first_name, last_name, username, billing, shipping, avatar_url, customer_note), names only.")
	CustomerDeleted = defineAction("customers.customer.deleted",
		"A customer was deleted. data: woo_customer_id.")

	// A company customer's contact persons; their names, email and phone are
	// personal data and stay out of the events.
	CustomerContactPersonAdded = defineAction("customers.contact_person.added",
		"A contact person was added to a company customer. data: customer_id.")
	CustomerContactPersonUpdated = defineAction("customers.contact_person.updated",
		"A contact person changed. data: customer_id, changed_fields (names only).")
	CustomerContactPersonRemoved = defineAction("customers.contact_person.removed",
		"A contact person was removed. data: customer_id.")

	// A customer's shipping addresses; the address itself is personal data
	// and stays out of the events.
	CustomerShippingAddressAdded = defineAction("customers.shipping_address.added",
		"A shipping address was added to a customer. data: customer_id, default (whether it is the default now).")
	CustomerShippingAddressUpdated = defineAction("customers.shipping_address.updated",
		"A shipping address changed. data: customer_id, changed_fields (names only).")
	CustomerShippingAddressRemoved = defineAction("customers.shipping_address.removed",
		"A shipping address was removed. data: customer_id.")
	CustomerShippingAddressDefaultChanged = defineAction("customers.shipping_address.default_changed",
		"Another of a customer's shipping addresses became the default (entity_id). data: customer_id.")
)
