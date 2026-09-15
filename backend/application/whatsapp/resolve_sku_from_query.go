package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ResolveSKUFromQueryUseCase turns free text (a SKU code or a product name,
// however the customer phrased it -- e.g. AIIntentParser's ProductQuery)
// into one concrete SKU code, for HandleInboundMessageUseCase to hand to
// the existing, unchanged LookupProductInfoUseCase (which does the real
// WhatsApp-channel-enrollment gating). Two steps, same order procurement's
// MatchExtractedLineItemsUseCase uses for its own text-to-SKU problem:
//  1. Exact SKU code match.
//  2. A fuzzy product-name match, but ONLY when it's unambiguous -- see
//     resolveByName's doc comment for the acceptance bar.
//
// Deliberately a standalone use case rather than an extension of
// MatchExtractedLineItemsUseCase.matchByName: that method is private to,
// and already covered by tests for, procurement's PO-line-item matching
// pipeline -- duplicating its (small, stable) scoring algorithm here avoids
// any risk of regressing tested code for an unrelated feature.
type ResolveSKUFromQueryUseCase struct {
	db          *gorm.DB
	skuRepo     pmsDomain.SKURepository
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
}

func NewResolveSKUFromQueryUseCase(
	db *gorm.DB, skuRepo pmsDomain.SKURepository, productRepo pmsDomain.ProductRepository, variantRepo pmsDomain.VariantRepository,
) *ResolveSKUFromQueryUseCase {
	return &ResolveSKUFromQueryUseCase{db: db, skuRepo: skuRepo, productRepo: productRepo, variantRepo: variantRepo}
}

// Execute returns "" (not an error) when query is empty or nothing resolves
// -- the caller (buildReply) treats that exactly like today's "unknown SKU
// code" outcome, a normal reply, not a dispatch error.
func (uc *ResolveSKUFromQueryUseCase) Execute(ctx context.Context, query string) (string, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return "", nil
	}

	var skuCode string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if code, err := uc.resolveExact(ctx, trimmed); err != nil {
			return err
		} else if code != "" {
			skuCode = code
			return nil
		}
		code, err := uc.resolveByName(ctx, trimmed)
		if err != nil {
			return err
		}
		skuCode = code
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("resolveSKUFromQuery: %w", err)
	}
	return skuCode, nil
}

// resolveExact tries the query as a literal SKU code, as typed and
// uppercased (every SKU code in this catalog is uppercase by convention,
// but a customer -- or a model normalizing their text -- may not type it
// that way).
func (uc *ResolveSKUFromQueryUseCase) resolveExact(ctx context.Context, query string) (string, error) {
	for _, candidate := range uniqueNonEmpty(query, strings.ToUpper(query)) {
		sku, err := uc.skuRepo.FindByCode(ctx, candidate)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				continue
			}
			return "", err
		}
		return sku.SKUCode, nil
	}
	return "", nil
}

// resolveByName is the fuzzy fallback -- see nameMatchMinScore/
// nameMatchMinMargin's doc comment for the acceptance bar. Only resolves
// when the accepted product also has exactly one SKU across all its
// variants (a multi-variant product can't be safely narrowed to one SKU by
// name text alone) -- returns "", nil otherwise, same "leave it, don't
// guess" convention as procurement's matchByName.
func (uc *ResolveSKUFromQueryUseCase) resolveByName(ctx context.Context, query string) (string, error) {
	tokens := significantTokens(query)
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
	if len(skus) != 1 {
		return "", nil
	}
	return skus[0].SKUCode, nil
}

func uniqueNonEmpty(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
