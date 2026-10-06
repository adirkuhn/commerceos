package events

// Catalog (pms) actions not already covered by a typed pms.* event
// (backend/application/pms).
var (
	CategoryCreated = defineAction("pms.category.created",
		"A product category was created. data: name, slug, parent_id.")
	CategoryDeleted = defineAction("pms.category.deleted",
		"A product category was deleted. data: name.")
	MediaUploaded = defineAction("pms.media.uploaded",
		"An image was uploaded for a product or variant. data: product_id, variant_id, url, position.")
	MediaDeleted = defineAction("pms.media.deleted",
		"A product or variant image was deleted (its file removed from storage). data: product_id, variant_id.")
	SKUCurrencyPriceSaved = defineAction("pms.sku_currency_price.saved",
		"A SKU's fixed price in another currency was set or changed (entity_id is the SKU; changes: price_minor_units, sale_price_minor_units, sale_end_date). data: currency_code.")
	SKUCurrencyPriceDeleted = defineAction("pms.sku_currency_price.deleted",
		"A SKU's fixed price in another currency was removed -- it falls back to exchange-rate conversion (entity_id is the SKU). data: currency_code.")
	SKUCostPriceChanged = defineAction("pms.sku.cost_price_changed",
		"A SKU's cost price changed (changes: cost_price_minor_units, cost_price_source). data: reason -- receipt (stock received at a landed cost, moving average), correction (a value correction such as a corrected invoice), manual (set by hand, overrides computed cost) or override_cleared (back to the computed cost); reference_type, reference_id.")
	BarcodeAliasCreated = defineAction("pms.barcode_alias.created",
		"An extra barcode (e.g. a case barcode) was linked to a SKU. data: sku_id, code, units_per_scan.")
	BarcodeAliasDeleted = defineAction("pms.barcode_alias.deleted",
		"An extra barcode was unlinked from its SKU.")
	ScaleCodeAliasCreated = defineAction("pms.scale_code_alias.created",
		"An extra scale code (a second weighing scale's PLU) was linked to a SKU. data: sku_id, code.")
	ScaleCodeAliasDeleted = defineAction("pms.scale_code_alias.deleted",
		"An extra scale code was unlinked from its SKU.")
	ProductDuplicated = defineAction("pms.product.duplicated",
		"A product was copied (as a draft) with its variants and SKUs; entity_id is the new product. data: source_product_id, variant_count, sku_count.")
)
