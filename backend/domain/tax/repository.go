package tax

import "context"

type TaxClassRepository interface {
	Create(ctx context.Context, c *TaxClass) error
	FindByID(ctx context.Context, id string) (*TaxClass, error)
	// List returns every tax class for the tenant, in no particular order
	// -- small, tenant-scoped set, callers sort/group client-side if needed.
	List(ctx context.Context) ([]TaxClass, error)
	Update(ctx context.Context, c *TaxClass) error
	Delete(ctx context.Context, id string) error
}

type TaxRateRepository interface {
	// Upsert inserts or updates by the (tenant_id, tax_class_id, country)
	// unique constraint -- a rate is identified by what it applies to, not
	// a caller-supplied ID, so there's no separate Create/Update split.
	Upsert(ctx context.Context, r *TaxRate) error
	FindByID(ctx context.Context, id string) (*TaxRate, error)
	// List returns every rate for the tenant across all tax classes --
	// small dataset, callers (both the admin UI and tax-calculation
	// client-side logic) filter/group by TaxClassID themselves.
	List(ctx context.Context) ([]TaxRate, error)
	Delete(ctx context.Context, id string) error
}

type ExternalTaxClassLinkRepository interface {
	// Upsert inserts or updates by the (tenant_id, provider, external_slug)
	// unique constraint -- same "identified by what it applies to" shape
	// as TaxRateRepository.Upsert. Used both by import (auto-create on
	// first sight of a new slug) and by staff repointing an existing link
	// to a different TaxClass.
	Upsert(ctx context.Context, l *ExternalTaxClassLink) error
	FindByProviderAndSlug(ctx context.Context, provider, externalSlug string) (*ExternalTaxClassLink, error)
	// List returns every link for the tenant across all providers -- small
	// dataset, same rationale as TaxRateRepository.List.
	List(ctx context.Context) ([]ExternalTaxClassLink, error)
	Delete(ctx context.Context, id string) error
}
