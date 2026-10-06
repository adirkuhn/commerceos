package channels

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"strings"
	"time"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// reconcileWooTaxClasses maps each distinct Woo tax_class slug found in the
// store's rate table onto one of this tenant's domain/tax.TaxClass rows --
// reusing the link recorded by an earlier run (or set by staff in Settings
// > Tax) when one exists, auto-creating both the TaxClass and its link the
// first time a new slug is seen. Country-level rates (state == "") for
// that class are upserted from Woo's own numbers. Returns early without
// touching the DB at all when the store reports no tax rates -- keeps a
// tenant that hasn't configured Woo tax from paying for an empty
// transaction on every import run.
func (uc *ImportWooProductsUseCase) reconcileWooTaxClasses(ctx context.Context, wooRates []integrationsDomain.WooTaxRate) (map[string]string, error) {
	ratesByClass := make(map[string][]integrationsDomain.WooTaxRate)
	for _, r := range wooRates {
		ratesByClass[r.Class] = append(ratesByClass[r.Class], r)
	}
	if len(ratesByClass) == 0 {
		return map[string]string{}, nil
	}

	slugToClassID := make(map[string]string, len(ratesByClass))
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for slug, rates := range ratesByClass {
			taxClassID, err := uc.resolveOrCreateTaxClassForWooSlug(ctx, slug)
			if err != nil {
				return err
			}
			slugToClassID[slug] = taxClassID

			for _, r := range rates {
				// Country-level only -- see domain/tax.TaxRate's scope
				// (no state/postcode granularity). A row with no country
				// at all ("applies everywhere") has nothing meaningful to
				// key our tenant-wide-per-country table on, so it's
				// skipped rather than guessed at.
				if r.State != "" || r.Country == "" {
					continue
				}
				bp, err := ParseWooPriceToMinorUnits(r.Rate)
				if err != nil {
					continue // an unparseable single rate row shouldn't abort the whole reconciliation
				}
				now := time.Now()
				rate := taxDomain.TaxRate{
					TenantID: tenant.IDFromContext(ctx), TaxClassID: taxClassID,
					Country: strings.ToUpper(r.Country), RateBasisPoints: int32(bp),
					CreatedAt: now, UpdatedAt: now,
				}
				if err := uc.taxRateRepo.Upsert(ctx, &rate); err != nil {
					return err
				}
				if err := eventrecord.Record(ctx, uc.outbox, events.ChannelTaxRateImported, rate.ID, eventrecord.Data{
					"provider": string(integrationsDomain.ProviderWooCommerce), "tax_class_id": rate.TaxClassID,
					"country": rate.Country, "rate_basis_points": rate.RateBasisPoints,
				}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	return slugToClassID, err
}

// resolveOrCreateTaxClassForWooSlug looks up an existing link for this Woo
// tax_class slug; if none exists yet, creates a new TaxClass (named from
// the slug -- see HumanizeWooTaxClassSlug) and records the link so this
// and future runs, and Settings > Tax's external-links list, agree on it.
func (uc *ImportWooProductsUseCase) resolveOrCreateTaxClassForWooSlug(ctx context.Context, slug string) (string, error) {
	link, err := uc.taxLinkRepo.FindByProviderAndSlug(ctx, string(integrationsDomain.ProviderWooCommerce), slug)
	if err == nil {
		return link.TaxClassID, nil
	}
	if !errors.Is(err, taxDomain.ErrExternalTaxClassLinkNotFound) {
		return "", err
	}

	now := time.Now()
	class := taxDomain.TaxClass{
		TenantID: tenant.IDFromContext(ctx), Name: HumanizeWooTaxClassSlug(slug), CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.taxClassRepo.Create(ctx, &class); err != nil {
		return "", err
	}
	newLink := taxDomain.ExternalTaxClassLink{
		TenantID: tenant.IDFromContext(ctx), Provider: string(integrationsDomain.ProviderWooCommerce),
		ExternalSlug: slug, TaxClassID: class.ID, CreatedAt: now, UpdatedAt: now,
	}
	if err := uc.taxLinkRepo.Upsert(ctx, &newLink); err != nil {
		return "", err
	}
	if err := eventrecord.Record(ctx, uc.outbox, events.ChannelTaxClassImported, class.ID, eventrecord.Data{
		"provider": newLink.Provider, "external_slug": slug, "name": class.Name,
	}); err != nil {
		return "", err
	}
	return class.ID, nil
}

// resolveProductTaxClassID resolves one product/variation's Woo tax_class
// slug to a TaxClassID -- first from this run's reconciled map (the common
// case), falling back to a direct link lookup for a slug the store's rate
// table didn't mention at all (a class with zero configured rates). Never
// errors: an unresolvable slug just means the SKU falls back to the
// tenant's default class, same as before this feature existed.
func (uc *ImportWooProductsUseCase) resolveProductTaxClassID(ctx context.Context, slugToClassID map[string]string, slug string) *string {
	if id, ok := slugToClassID[slug]; ok {
		return &id
	}
	link, err := uc.taxLinkRepo.FindByProviderAndSlug(ctx, string(integrationsDomain.ProviderWooCommerce), slug)
	if err != nil {
		return nil
	}
	id := link.TaxClassID
	return &id
}
