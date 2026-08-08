package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

type MatchSource string

const (
	MatchSourceExactSKU  MatchSource = "exact_sku"
	MatchSourceAlias     MatchSource = "alias"
	MatchSourceName      MatchSource = "name"
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
// SupplierProductAlias from a past human correction, (3) a fuzzy product-name
// match when the first two find nothing. Name matching can't be a simple
// substring check either direction: real document text and a catalog's
// canonical product name routinely fail to contain each other verbatim (e.g.
// doc "Chá Limão, Gengibre e Mel BARÃO 10 saq." vs catalog "Cha de Limão,
// Gengibre e Mel - Barão de Cotegipe" -- different word order, a pack-size
// suffix on one side, a brand suffix on the other). Instead it tokenizes both
// strings and scores word overlap (see nameMatchScore) -- see matchByName's
// doc comment for the full algorithm and its safety thresholds. Name search
// only resolves a match when it is unambiguous: a confident, clearly-best
// scoring product, that also has exactly one SKU total across all its
// variants -- a product with several variants (size/color/etc.) can't be
// safely auto-picked down to one SKU from name text alone, so it's left for
// the review UI instead, same as any other unmatched item. A fuzzier step
// still -- asking the AI to pick from a shortlist of candidate products -- is
// deliberately not implemented: AIDocumentParser's ExtractLineItems is a
// whole-document extraction call, not a per-item classifier, and bolting
// that on speculatively would be a bigger, under-tested feature on its own.
// Anything that falls through all three steps is left unmatched for the
// review UI (see link_line_item_to_sku.go), same as the plan's
// needs_review path.
type MatchExtractedLineItemsUseCase struct {
	skuRepo     pmsDomain.SKURepository
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	aliasRepo   procurementDomain.SupplierProductAliasRepository
}

func NewMatchExtractedLineItemsUseCase(
	skuRepo pmsDomain.SKURepository,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	aliasRepo procurementDomain.SupplierProductAliasRepository,
) *MatchExtractedLineItemsUseCase {
	return &MatchExtractedLineItemsUseCase{skuRepo: skuRepo, productRepo: productRepo, variantRepo: variantRepo, aliasRepo: aliasRepo}
}

// Execute must be called with a ctx that already carries a transaction (via
// tenant.WithTenant) -- same convention as the repos it calls, since it's a
// read-only composite step meant to run as part of a larger job, not a
// standalone top-level use case.
func (uc *MatchExtractedLineItemsUseCase) Execute(ctx context.Context, supplierID string, extracted []procurementDomain.ExtractedLineItem) ([]MatchedLineItem, error) {
	matched := make([]MatchedLineItem, 0, len(extracted))
	for _, item := range extracted {
		m := MatchedLineItem{Extracted: item, MatchSource: MatchSourceUnmatched}

		// A charge row (shipping/discount/fee/credit note) isn't a product
		// -- there's nothing to match it against, and it must never be
		// left "unmatched" waiting for a SKU link that will never come.
		if !item.LineType.IsProduct() {
			matched = append(matched, m)
			continue
		}

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

		if m.SKUID == nil {
			id, err := uc.matchByName(ctx, item.RawDescription)
			if err != nil {
				return nil, fmt.Errorf("matchExtractedLineItems: %w", err)
			}
			if id != "" {
				m.SKUID = &id
				m.MatchSource = MatchSourceName
			}
		}

		matched = append(matched, m)
	}
	return matched, nil
}

// nameMatchMinScore and nameMatchMinMargin are matchByName's acceptance
// thresholds -- see its doc comment.
const (
	nameMatchMinScore  = 0.6
	nameMatchMinMargin = 0.15
)

// matchByName is the matching pipeline's third and final step (see
// MatchExtractedLineItemsUseCase's doc comment for the containment-search
// approach this replaced and why). It tokenizes rawDescription, asks
// ProductRepository.SearchByAnyNameToken for every product whose name
// contains at least one of those tokens (a deliberately wide net -- real
// candidates routinely differ in word order/suffixes from either side), then
// scores each candidate in Go with nameMatchScore. A candidate is only
// accepted as a match when both hold:
//   - its score is at least nameMatchMinScore (most of the product's own
//     name words are found in the document text)
//   - it beats the runner-up by at least nameMatchMinMargin (not a
//     coin-flip between two similarly-named products)
//
// Returns "" (not an error) whenever no candidate clears both bars, or the
// accepted candidate doesn't have exactly one SKU across all its variants --
// a multi-variant product can't be safely narrowed to one SKU by name text
// alone, so it's left for human review same as any other unmatched item.
func (uc *MatchExtractedLineItemsUseCase) matchByName(ctx context.Context, rawDescription string) (string, error) {
	tokens := significantTokens(rawDescription)
	if len(tokens) == 0 {
		return "", nil
	}

	candidates, err := uc.productRepo.SearchByAnyNameToken(ctx, tokens, 50)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", nil
	}

	best := -1
	bestScore, secondScore := -1.0, -1.0
	for i, p := range candidates {
		score := nameMatchScore(p.Name, tokens)
		if score > bestScore {
			best, bestScore, secondScore = i, score, bestScore
		} else if score > secondScore {
			secondScore = score
		}
	}
	if best == -1 || bestScore < nameMatchMinScore || bestScore-secondScore < nameMatchMinMargin {
		return "", nil
	}

	variants, err := uc.variantRepo.ListByProduct(ctx, candidates[best].ID)
	if err != nil {
		return "", err
	}

	var skus []pmsDomain.SKU
	for _, v := range variants {
		vSKUs, err := uc.skuRepo.ListByVariant(ctx, v.ID)
		if err != nil {
			return "", err
		}
		skus = append(skus, vSKUs...)
	}
	// More than one SKU (multi-variant product) or none at all -- either way
	// name text alone can't safely pick one, leave it for human review.
	if len(skus) != 1 {
		return "", nil
	}
	return skus[0].ID, nil
}

// nameMatchScore returns what fraction of productName's own significant
// words are also present in docTokens -- e.g. catalog "Cha de Limão,
// Gengibre e Mel - Barão de Cotegipe" against a document's ["cha", "limao",
// "gengibre", "mel", "barao", "10", "saq"] scores 5/6 (only "cotegipe" is
// missing), while an unrelated "Cha de Camomila - Barão de Cotegipe" scores
// low against the same tokens. Deliberately asymmetric (product-name
// coverage, not a symmetric Jaccard score): document text commonly carries
// extra pack-size/quantity words a catalog name doesn't have, and a catalog
// name commonly carries an extra brand/supplier suffix the document doesn't
// -- penalizing either side equally would punish exactly the normal case
// this step exists for. 0 when productName has no significant words.
func nameMatchScore(productName string, docTokens []string) float64 {
	productTokens := significantTokens(productName)
	if len(productTokens) == 0 {
		return 0
	}
	docSet := make(map[string]struct{}, len(docTokens))
	for _, t := range docTokens {
		docSet[t] = struct{}{}
	}
	overlap := 0
	for _, t := range productTokens {
		if _, ok := docSet[t]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(productTokens))
}

// nameMatchStopwords are short connector words common in both Portuguese and
// English product names/descriptions -- excluded so they don't inflate
// nameMatchScore's overlap count with matches that carry no actual product
// identity (e.g. "e"/"and", "de"/"of").
var nameMatchStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "de": {}, "da": {}, "do": {},
	"das": {}, "dos": {}, "em": {}, "para": {}, "com": {}, "por": {},
}

// significantTokens lowercases s, strips diacritics (manual Latin mapping --
// no new dependency for this, see stripDiacritics), replaces every run of
// non-alphanumeric characters with a single space, then splits into words,
// dropping stopwords and anything under 3 runes (which also removes bare
// pack-size/quantity digits like "10" -- noise for name identity, not signal
// for it).
func significantTokens(s string) []string {
	normalized := stripDiacritics(strings.ToLower(s))
	fields := strings.FieldsFunc(normalized, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if len(f) < 3 {
			continue
		}
		if _, stop := nameMatchStopwords[f]; stop {
			continue
		}
		tokens = append(tokens, f)
	}
	return tokens
}

// diacriticsReplacer covers the accented Latin letters that actually appear
// in this catalog's product names (Portuguese, plus common Spanish/French
// overlap) -- a manual table instead of pulling in a Unicode-normalization
// package for what's otherwise a small, fixed alphabet.
var diacriticsReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

func stripDiacritics(s string) string {
	return diacriticsReplacer.Replace(s)
}

// resyncAliasMatchedLineItems re-checks a PO's line items against the given
// supplier's learned aliases and returns whether it changed anything. Two
// calling situations, same logic either way:
//
//   - Supplier just confirmed on a needs_review PO: the parse-time matching
//     pass (MatchExtractedLineItemsUseCase.Execute) ran before the PO had a
//     supplier -- an AI-uploaded document's normal case, since the supplier
//     is only assigned afterwards via the review flow -- so its alias
//     lookup had no supplierID to check against and left these unmatched.
//   - Supplier changed to a different one: any item still carrying an
//     unreviewed alias guess from the old supplier needs to be re-evaluated
//     against the new supplier's aliases, not left showing a stale guess.
//
// Only items with no confirmed link, or whose current link itself came from
// an unreviewed alias guess (SKUAutoLinkedFromAlias), are touched -- an
// exact-SKU match or an explicit human link (see LinkLineItemToSKUUseCase,
// UpdatePurchaseOrderLineItemUseCase) is never revisited here.
func resyncAliasMatchedLineItems(
	ctx context.Context,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	aliasRepo procurementDomain.SupplierProductAliasRepository,
	purchaseOrderID, supplierID string,
) (bool, error) {
	items, err := lineItemRepo.ListByPurchaseOrder(ctx, purchaseOrderID)
	if err != nil {
		return false, fmt.Errorf("resyncAliasMatchedLineItems: %w", err)
	}
	changed := false
	for _, item := range items {
		// A charge row never gets a SKU, alias-sourced or otherwise.
		if !item.LineType.IsProduct() {
			continue
		}
		if item.SKUID != nil && !item.SKUAutoLinkedFromAlias {
			continue
		}

		var newSKUID *string
		if rawKey := NormalizeRawKey(item.RawSKU, item.RawDescription); rawKey != "" {
			alias, err := aliasRepo.FindBySupplierAndRawKey(ctx, supplierID, rawKey)
			if err != nil {
				return false, fmt.Errorf("resyncAliasMatchedLineItems: %w", err)
			}
			if alias != nil {
				id := alias.SKUID
				newSKUID = &id
			}
		}

		if item.SKUID == nil && newSKUID == nil {
			continue
		}
		if item.SKUID != nil && newSKUID != nil && *item.SKUID == *newSKUID {
			continue
		}

		item.SKUID = newSKUID
		item.SKUAutoLinkedFromAlias = newSKUID != nil
		item.UpdatedAt = time.Now()
		if err := lineItemRepo.Update(ctx, &item); err != nil {
			return false, fmt.Errorf("resyncAliasMatchedLineItems: %w", err)
		}
		changed = true
	}
	return changed, nil
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
