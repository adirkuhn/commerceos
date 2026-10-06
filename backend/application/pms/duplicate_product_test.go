package pms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// stubDupProductRepo/stubDupVariantRepo/stubDupSKURepo are local to this
// file -- the package's existing stubProductRepo/stubVariantRepo (see
// create_variant_test.go) hardcode FindBySlug/FindByCode to always report
// "not found", which can't exercise the collision-retry paths this use
// case needs covered.

type stubDupProductRepo struct {
	pmsDomain.ProductRepository
	source      *pmsDomain.Product
	findByIDErr error
	takenSlugs  map[string]bool
	created     []pmsDomain.Product
}

func (s *stubDupProductRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Product, error) {
	if s.findByIDErr != nil {
		return nil, s.findByIDErr
	}
	return s.source, nil
}
func (s *stubDupProductRepo) FindBySlug(_ context.Context, slug string) (*pmsDomain.Product, error) {
	if s.takenSlugs[slug] {
		return &pmsDomain.Product{Slug: slug}, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubDupProductRepo) Create(_ context.Context, p *pmsDomain.Product) error {
	p.ID = "new-product-1"
	s.created = append(s.created, *p)
	return nil
}

type stubDupVariantRepo struct {
	pmsDomain.VariantRepository
	source  []pmsDomain.Variant
	created []pmsDomain.Variant
}

func (s *stubDupVariantRepo) ListByProduct(_ context.Context, _ string) ([]pmsDomain.Variant, error) {
	return s.source, nil
}
func (s *stubDupVariantRepo) Create(_ context.Context, v *pmsDomain.Variant) error {
	v.ID = "new-variant-1"
	s.created = append(s.created, *v)
	return nil
}

type stubDupSKURepo struct {
	pmsDomain.SKURepository
	byVariant  map[string][]pmsDomain.SKU
	takenCodes map[string]bool
	created    []pmsDomain.SKU
}

func (s *stubDupSKURepo) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return s.byVariant[variantID], nil
}
func (s *stubDupSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if s.takenCodes[code] {
		return &pmsDomain.SKU{SKUCode: code}, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubDupSKURepo) Create(_ context.Context, sk *pmsDomain.SKU) error {
	sk.ID = "new-sku-1"
	s.created = append(s.created, *sk)
	return nil
}

func TestDuplicateProduct_clonesProductVariantsAndSKUsAsDraft(t *testing.T) {
	sourceCategoryID := "cat-1"
	source := &pmsDomain.Product{
		ID: "source-1", Name: "T-Shirt", CategoryID: &sourceCategoryID, Description: "desc",
		Status: pmsDomain.ProductStatusActive, HasVariants: true, Tags: []string{"summer"},
		SyncVariantsAsWooVariations: true,
	}
	sourceBarcode := "0123456789"
	costPrice := int64(500)
	sourceVariant := pmsDomain.Variant{ID: "variant-1", ProductID: "source-1", Name: "Large", Position: 1}
	sourceSKU := pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", SKUCode: "TSHIRT-L", Barcode: sourceBarcode,
		PriceMinorUnits: 1999, CurrencyCode: "USD", CostPriceMinorUnits: &costPrice,
		CostPriceSource: pmsDomain.SKUCostPriceSourceComputed, Status: pmsDomain.SKUStatusActive,
	}

	productRepo := &stubDupProductRepo{source: source, takenSlugs: map[string]bool{}}
	variantRepo := &stubDupVariantRepo{source: []pmsDomain.Variant{sourceVariant}}
	skuRepo := &stubDupSKURepo{byVariant: map[string][]pmsDomain.SKU{"variant-1": {sourceSKU}}, takenCodes: map[string]bool{}}
	uc := apppms.NewDuplicateProductUseCase(newMockGormDB(t), productRepo, variantRepo, skuRepo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), "source-1")

	require.NoError(t, err)
	require.Len(t, productRepo.created, 1)
	assert.Equal(t, "T-Shirt (Copy)", productRepo.created[0].Name)
	assert.Equal(t, "t-shirt-copy", productRepo.created[0].Slug)
	assert.Equal(t, pmsDomain.ProductStatusDraft, productRepo.created[0].Status, "duplicate must always be draft, regardless of the source's status")
	assert.Equal(t, &sourceCategoryID, productRepo.created[0].CategoryID)
	assert.Equal(t, []string{"summer"}, productRepo.created[0].Tags)

	require.Len(t, variantRepo.created, 1)
	assert.Equal(t, "new-product-1", variantRepo.created[0].ProductID)
	assert.Equal(t, "Large", variantRepo.created[0].Name)

	require.Len(t, skuRepo.created, 1)
	dup := skuRepo.created[0]
	assert.Equal(t, "new-variant-1", dup.VariantID)
	assert.Equal(t, "TSHIRT-L-COPY", dup.SKUCode)
	assert.Equal(t, "", dup.Barcode, "barcode must not be copied -- it identifies a physical item")
	assert.Equal(t, int64(1999), dup.PriceMinorUnits)
	assert.Nil(t, dup.CostPriceMinorUnits, "cost price must not be copied -- no purchase history for the duplicate yet")

	assert.Equal(t, "new-product-1", result.Product.ID)
	require.Len(t, result.Variants, 1)
	require.Len(t, result.Variants[0].SKUs, 1)
}

func TestDuplicateProduct_neverRaisesProductChanged(t *testing.T) {
	// DuplicateProductUseCase structurally cannot enqueue pms.product.changed
	// -- it takes no outbox.Repository dependency at all. This test is a
	// canary should that ever change: NewDuplicateProductUseCase's
	// signature is exactly (db, productRepo, variantRepo, skuRepo); adding
	// an outboxRepo param would fail to compile here, forcing a conscious
	// decision, not a silent regression that re-enables auto-enrollment
	// (see EnrollProductInChannelsUseCase) for duplicated products.
	source := &pmsDomain.Product{ID: "source-1", Name: "Widget", Status: pmsDomain.ProductStatusActive}
	productRepo := &stubDupProductRepo{source: source, takenSlugs: map[string]bool{}}
	variantRepo := &stubDupVariantRepo{}
	skuRepo := &stubDupSKURepo{}
	_ = apppms.NewDuplicateProductUseCase(newMockGormDB(t), productRepo, variantRepo, skuRepo, &eventrecordtest.Outbox{})
}

func TestDuplicateProduct_retriesNameOnSlugCollision(t *testing.T) {
	source := &pmsDomain.Product{ID: "source-1", Name: "Widget", Status: pmsDomain.ProductStatusActive}
	productRepo := &stubDupProductRepo{source: source, takenSlugs: map[string]bool{"widget-copy": true}}
	variantRepo := &stubDupVariantRepo{}
	skuRepo := &stubDupSKURepo{}
	uc := apppms.NewDuplicateProductUseCase(newMockGormDB(t), productRepo, variantRepo, skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "source-1")

	require.NoError(t, err)
	require.Len(t, productRepo.created, 1)
	assert.Equal(t, "Widget (Copy 2)", productRepo.created[0].Name)
	assert.Equal(t, "widget-copy-2", productRepo.created[0].Slug)
}

func TestDuplicateProduct_retriesSKUCodeOnCollision(t *testing.T) {
	source := &pmsDomain.Product{ID: "source-1", Name: "Widget", Status: pmsDomain.ProductStatusActive}
	sourceVariant := pmsDomain.Variant{ID: "variant-1", Name: "Default"}
	sourceSKU := pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1", SKUCode: "WID-1", PriceMinorUnits: 1000, CurrencyCode: "USD"}

	productRepo := &stubDupProductRepo{source: source, takenSlugs: map[string]bool{}}
	variantRepo := &stubDupVariantRepo{source: []pmsDomain.Variant{sourceVariant}}
	skuRepo := &stubDupSKURepo{byVariant: map[string][]pmsDomain.SKU{"variant-1": {sourceSKU}}, takenCodes: map[string]bool{"WID-1-COPY": true}}
	uc := apppms.NewDuplicateProductUseCase(newMockGormDB(t), productRepo, variantRepo, skuRepo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "source-1")

	require.NoError(t, err)
	require.Len(t, skuRepo.created, 1)
	assert.Equal(t, "WID-1-COPY-2", skuRepo.created[0].SKUCode)
}

func TestDuplicateProduct_propagatesSourceNotFound(t *testing.T) {
	productRepo := &stubDupProductRepo{findByIDErr: pmsDomain.ErrProductNotFound}
	uc := apppms.NewDuplicateProductUseCase(newMockGormDB(t), productRepo, &stubDupVariantRepo{}, &stubDupSKURepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	assert.True(t, errors.Is(err, pmsDomain.ErrProductNotFound))
}
