package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

type MatchSource string

const (
	MatchSourceExactSKU  MatchSource = "exact_sku"
	MatchSourceAlias     MatchSource = "alias"
	MatchSourceUnmatched MatchSource = "unmatched"
)

// MatchedLineItem is one ExtractedLineItem after running it through the
// matching pipeline -- SKUID is nil when MatchSource is Unmatched.
type MatchedLineItem struct {
	Extracted   procurementDomain.ExtractedLineItem
	SKUID       *string
	MatchSource MatchSource
}

// MatchExtractedLineItemsUseCase implements the matching pipeline described
// in the plan: (1) exact SKURepository.FindByCode match, (2) a learned
// SupplierProductAlias from a past human correction. A third step --
// asking the AI to pick from a shortlist of candidate products -- is
// deliberately not implemented yet: AIDocumentParser's ExtractLineItems is
// a whole-document extraction call, not a per-item classifier, and bolting
// that on speculatively would be a bigger, under-tested feature on its own.
// Anything that falls through both steps is left unmatched for the review
// UI (see link_line_item_to_sku.go), same as the plan's needs_review path.
type MatchExtractedLineItemsUseCase struct {
	skuRepo   pmsDomain.SKURepository
	aliasRepo procurementDomain.SupplierProductAliasRepository
}

func NewMatchExtractedLineItemsUseCase(skuRepo pmsDomain.SKURepository, aliasRepo procurementDomain.SupplierProductAliasRepository) *MatchExtractedLineItemsUseCase {
	return &MatchExtractedLineItemsUseCase{skuRepo: skuRepo, aliasRepo: aliasRepo}
}

// Execute must be called with a ctx that already carries a transaction (via
// tenant.WithTenant) -- same convention as the repos it calls, since it's a
// read-only composite step meant to run as part of a larger job, not a
// standalone top-level use case.
func (uc *MatchExtractedLineItemsUseCase) Execute(ctx context.Context, supplierID string, extracted []procurementDomain.ExtractedLineItem) ([]MatchedLineItem, error) {
	matched := make([]MatchedLineItem, 0, len(extracted))
	for _, item := range extracted {
		m := MatchedLineItem{Extracted: item, MatchSource: MatchSourceUnmatched}

		if item.RawSKU != "" {
			sku, err := uc.skuRepo.FindByCode(ctx, item.RawSKU)
			if err != nil && !errors.Is(err, pmsDomain.ErrSKUNotFound) {
				return nil, fmt.Errorf("matchExtractedLineItems: %w", err)
			}
			if sku != nil {
				id := sku.ID
				m.SKUID = &id
				m.MatchSource = MatchSourceExactSKU
			}
		}

		// No supplier means no alias to look up -- supplierID is a UUID
		// column on the other end, so an empty string would fail the query
		// outright rather than just miss (see
		// CreatePurchaseOrderFromDocumentUseCase's supplier-optional upload).
		if m.SKUID == nil && supplierID != "" {
			rawKey := NormalizeRawKey(item.RawSKU, item.RawDescription)
			if rawKey != "" {
				alias, err := uc.aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, rawKey)
				if err != nil {
					return nil, fmt.Errorf("matchExtractedLineItems: %w", err)
				}
				if alias != nil {
					id := alias.SKUID
					m.SKUID = &id
					m.MatchSource = MatchSourceAlias
				}
			}
		}

		matched = append(matched, m)
	}
	return matched, nil
}

// NormalizeRawKey produces the lookup key used for both writing and reading
// SupplierProductAlias.RawKey -- prefers the raw SKU/barcode text (more
// specific and more likely to repeat verbatim across a supplier's
// documents) and falls back to the description.
func NormalizeRawKey(rawSKU, rawDescription string) string {
	key := rawSKU
	if key == "" {
		key = rawDescription
	}
	return strings.ToLower(strings.TrimSpace(key))
}
