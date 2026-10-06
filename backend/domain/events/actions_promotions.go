package events

// Promotion actions (backend/application/promotions). A promotion's lines
// are part of it: "lines" lists {sku_id, batch_id, discount_type,
// discount_value} per line.
var (
	PromotionCreated = defineAction("promotions.promotion.created",
		"A promotion was created. data: name, type, status, starts_at, ends_at, lines.")
	PromotionUpdated = defineAction("promotions.promotion.updated",
		"A promotion changed (changes: name, type, status, starts_at, ends_at, lines). data.reason clearance_finished when the system ended it because its last clearance line finished.")
	PromotionDeleted = defineAction("promotions.promotion.deleted",
		"A promotion was deleted (applied prices were reverted first, each raising promotion_line.reverted). data: name.")
	PromotionLineApplied = defineAction("promotions.promotion_line.applied",
		"A promotion line put its SKU on sale. data: promotion_id, sku_id, sale_price_minor_units, previous_sale_price_minor_units.")
	PromotionLineReverted = defineAction("promotions.promotion_line.reverted",
		"A promotion line's SKU went back to its pre-promotion sale price. data: promotion_id, sku_id, restored_sale_price_minor_units.")
	PromotionLineFinished = defineAction("promotions.promotion_line.finished",
		"A clearance line ended because its batch sold out, expired or was removed. data: promotion_id, sku_id, batch_id.")
)
