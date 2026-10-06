package events

// Tax actions (backend/application/tax).
var (
	TaxClassCreated = defineAction("tax.tax_class.created",
		"A tax class was created. data: name, is_default.")
	TaxClassUpdated = defineAction("tax.tax_class.updated",
		"A tax class changed (changes: name, is_default). data.reason default_moved when it lost is_default because another class became the default.")
	TaxClassDeleted = defineAction("tax.tax_class.deleted",
		"A tax class was deleted. data: name.")
	TaxRateSet = defineAction("tax.tax_rate.set",
		"A tax class's rate for a country was set. data: tax_class_id, country, rate_basis_points.")
	TaxRateDeleted = defineAction("tax.tax_rate.deleted",
		"A tax rate was removed. data: tax_class_id, country.")
	ExternalTaxClassLinkSet = defineAction("tax.external_tax_class_link.set",
		"A sales channel's tax class (e.g. a WooCommerce slug) was linked to a tax class. data: provider, external_slug, tax_class_id.")
	ExternalTaxClassLinkDeleted = defineAction("tax.external_tax_class_link.deleted",
		"A channel tax class link was removed. data: provider, external_slug, tax_class_id.")
)
