package events

// Supplier actions (backend/application/suppliers).
var (
	SupplierCreated = defineAction("suppliers.supplier.created",
		"A supplier was created. data: name, status.")
	SupplierUpdated = defineAction("suppliers.supplier.updated",
		"A supplier's details changed (changes: name, email, phone, country, lead_time_days, status, notes).")
	SupplierDeleted = defineAction("suppliers.supplier.deleted",
		"A supplier was deleted. data: name.")
)
