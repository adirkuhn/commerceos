package pms

import (
	"context"
	"errors"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateProductInput struct {
	ID                          string
	Name                        string
	Description                 string
	CategoryID                  *string
	Status                      pmsDomain.ProductStatus
	HasVariants                 bool
	SyncVariantsAsWooVariations bool
	Tags                        []string
	// Slug, when non-nil, explicitly changes the product's slug -- nil
	// leaves it untouched (the "never re-slug on rename" behavior below
	// still holds for Name changes). This is the only way to fix a
	// product stuck with a duplicate's "-copy" slug suffix. An empty
	// string regenerates the slug from Name rather than being rejected.
	Slug *string
}

type UpdateProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	listingRepo channelsDomain.ProductChannelListingRepository
	outboxRepo  outbox.Repository
	historyRepo pmsDomain.ProductHistoryRepository
}

func NewUpdateProductUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	outboxRepo outbox.Repository,
	historyRepo pmsDomain.ProductHistoryRepository,
) *UpdateProductUseCase {
	return &UpdateProductUseCase{db: db, productRepo: productRepo, variantRepo: variantRepo, listingRepo: listingRepo, outboxRepo: outboxRepo, historyRepo: historyRepo}
}

// Execute updates product fields. Slug is only changed when in.Slug is
// explicitly set -- re-slugging on every Name change would churn the
// product's canonical URL, so a bare rename never touches it. An explicitly
// empty slug regenerates from the (possibly also-changing) Name, same as
// CreateProductUseCase derives it -- an empty slug is never persisted.
func (uc *UpdateProductUseCase) Execute(ctx context.Context, in UpdateProductInput) (*pmsDomain.Product, error) {
	var product pmsDomain.Product
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.productRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		variants, err := uc.variantRepo.ListByProduct(ctx, in.ID)
		if err != nil {
			return err
		}
		if !in.HasVariants && len(variants) > 1 {
			return pmsDomain.ErrVariantsNotEnabled
		}
		if in.Slug != nil {
			regenerated := *in.Slug == ""
			slug := *in.Slug
			if regenerated {
				slug = Slugify(in.Name)
			}
			if slug != existing.Slug {
				if Slugify(slug) != slug {
					return pmsDomain.ErrInvalidSlug
				}
				if regenerated {
					// A regenerated slug carries no user intent about its
					// exact text -- unlike a slug the user typed themselves,
					// a collision here is auto-resolved with a numeric
					// suffix instead of bouncing the save back at them.
					resolved, err := uc.disambiguateSlug(ctx, slug, existing.ID)
					if err != nil {
						return err
					}
					slug = resolved
				} else {
					other, err := uc.productRepo.FindBySlug(ctx, slug)
					if err == nil && other.ID != existing.ID {
						return pmsDomain.ErrDuplicateSlug
					}
					if err != nil && !errors.Is(err, pmsDomain.ErrProductNotFound) {
						return err
					}
				}
				existing.Slug = slug
			}
		}
		wasHasVariants := existing.HasVariants
		oldName, oldStatus := existing.Name, existing.Status
		existing.Name = in.Name
		existing.Description = in.Description
		existing.CategoryID = in.CategoryID
		existing.Status = in.Status
		existing.HasVariants = in.HasVariants
		existing.SyncVariantsAsWooVariations = in.SyncVariantsAsWooVariations
		existing.Tags = in.Tags
		existing.UpdatedAt = time.Now()
		if err := uc.productRepo.Update(ctx, existing); err != nil {
			return err
		}
		product = *existing
		// Flipping a flat (!HasVariants) product into variants mode doesn't
		// touch its variant -- the same variant/SKU that already existed
		// keeps its own ID, just now addressed as "the first variant"
		// instead of the product's only one. Its channel listing rows,
		// though, were created back when the product was flat and are
		// keyed VariantID=nil (the "legacy single-SKU simple product"
		// case -- see ProductChannelListing.VariantID's own doc). Left
		// alone, the next push looks up existingByVariant[variant.ID],
		// finds nothing, and tries to CREATE the product/variation on Woo
		// again -- colliding with the one Woo already has under this exact
		// SKU. Relinking the existing row(s) to the variant's own ID here
		// avoids that collision entirely, without any Woo API call.
		if !wasHasVariants && in.HasVariants && len(variants) == 1 {
			if err := uc.relinkDefaultVariantListings(ctx, product.ID, variants[0].ID); err != nil {
				return err
			}
		}
		if err := recordProductHistoryFieldIfChanged(ctx, uc.historyRepo, pmsDomain.ProductHistoryEntityProduct, product.ID, pmsDomain.ProductHistoryFieldName, oldName, product.Name); err != nil {
			return err
		}
		if err := recordProductHistoryFieldIfChanged(ctx, uc.historyRepo, pmsDomain.ProductHistoryEntityProduct, product.ID, pmsDomain.ProductHistoryFieldStatus, string(oldStatus), string(product.Status)); err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, product.ID)
	})
	if err != nil {
		return nil, fmt.Errorf("updateProduct: %w", err)
	}
	return &product, nil
}

// relinkDefaultVariantListings repoints every product-level channel listing
// (VariantID nil) this product has to variantID -- see the call site's
// comment for why this only ever runs on a flat-to-variants transition.
func (uc *UpdateProductUseCase) relinkDefaultVariantListings(ctx context.Context, productID, variantID string) error {
	listings, err := uc.listingRepo.ListByProduct(ctx, productID)
	if err != nil {
		return err
	}
	for _, l := range listings {
		if l.VariantID != nil {
			continue
		}
		l := l
		vid := variantID
		l.VariantID = &vid
		if err := uc.listingRepo.Update(ctx, &l); err != nil {
			return err
		}
	}
	return nil
}

// disambiguateSlug finds the first of base, base-2, base-3, ... not already
// owned by a different product -- mirrors DuplicateProductUseCase's
// uniqueCopyName loop/bound, but suffixes only the slug, never the name
// (the name isn't being duplicated here, it just happens to slugify the
// same as another product's).
func (uc *UpdateProductUseCase) disambiguateSlug(ctx context.Context, base, ownProductID string) (string, error) {
	for i := 1; i <= maxDuplicateNameAttempts; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		other, err := uc.productRepo.FindBySlug(ctx, candidate)
		if errors.Is(err, pmsDomain.ErrProductNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		if other.ID == ownProductID {
			return candidate, nil
		}
	}
	return "", pmsDomain.ErrDuplicateSlug
}
