package promotions

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

// PromotionChangeReasonClearanceFinished: data.reason of a
// promotions.promotion.updated the system raised when the last clearance
// line finished and the promotion ended.
const PromotionChangeReasonClearanceFinished = "clearance_finished"

// lineSummaries is how a promotion's lines appear in its events.
func lineSummaries(lines []promotionsDomain.PromotionSKU) []map[string]any {
	out := make([]map[string]any, len(lines))
	for i, l := range lines {
		var batchID any
		if l.BatchID != nil {
			batchID = *l.BatchID
		}
		out[i] = map[string]any{
			"sku_id": l.SKUID, "batch_id": batchID,
			"discount_type": string(l.DiscountType), "discount_value": l.DiscountValue,
		}
	}
	return out
}

func promotionChanges(before, after promotionsDomain.Promotion, linesBefore, linesAfter []promotionsDomain.PromotionSKU) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("name", before.Name, after.Name).
		Set("type", string(before.Type), string(after.Type)).
		Set("status", string(before.Status), string(after.Status)).
		Set("starts_at", before.StartsAt, after.StartsAt).
		Set("ends_at", before.EndsAt, after.EndsAt).
		Set("lines", lineSummaries(linesBefore), lineSummaries(linesAfter))
}

func recordPromotionCreated(ctx context.Context, outbox eventrecord.Enqueuer, p promotionsDomain.Promotion, lines []promotionsDomain.PromotionSKU) error {
	return eventrecord.Record(ctx, outbox, events.PromotionCreated, p.ID, eventrecord.Data{
		"name": p.Name, "type": string(p.Type), "status": string(p.Status),
		"starts_at": p.StartsAt, "ends_at": p.EndsAt, "lines": lineSummaries(lines),
	})
}
