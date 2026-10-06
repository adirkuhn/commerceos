package events

// Customer actions (backend/application/customers). Personal data
// (name, email, phone, username, addresses, avatar, note) never travels in
// these events: changed personal fields are listed by name in
// data.changed_fields, without values -- read the customer by entity_id.
var (
	CustomerCreated = defineAction("customers.customer.created",
		"A customer was created. data: woo_customer_id, role, is_paying_customer.")
	CustomerUpdated = defineAction("customers.customer.updated",
		"A customer changed (changes: woo_customer_id, role, is_paying_customer). data.changed_fields: personal fields that changed (email, first_name, last_name, username, billing, shipping, avatar_url, customer_note), names only.")
	CustomerDeleted = defineAction("customers.customer.deleted",
		"A customer was deleted. data: woo_customer_id.")
)
