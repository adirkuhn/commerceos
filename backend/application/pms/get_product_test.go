package pms_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// fakeProductCatalog is products -> variants -> skus, found by id.
type fakeProductCatalog struct {
	products map[string]pmsDomain.Product
	variants map[string]pmsDomain.Variant
	skus     map[string]pmsDomain.SKU
}

func (f *fakeProductCatalog) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if p, ok := f.products[id]; ok {
		return &p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}

type fakeCatalogVariants struct{ *fakeProductCatalog }

func (f fakeCatalogVariants) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	var out []pmsDomain.Variant
	for _, v := range f.variants {
		if v.ProductID == productID {
			out = append(out, v)
		}
	}
	return out, nil
}
func (f fakeCatalogVariants) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	if v, ok := f.variants[id]; ok {
		return &v, nil
	}
	return nil, pmsDomain.ErrVariantNotFound
}

type fakeCatalogSKUs struct{ *fakeProductCatalog }

func (f fakeCatalogSKUs) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	var out []pmsDomain.SKU
	for _, s := range f.skus {
		if s.VariantID == variantID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (f fakeCatalogSKUs) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if s, ok := f.skus[id]; ok {
		return &s, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}

func strp(s string) *string { return &s }

// A case SKU (and a portion SKU) on "Caixa Refresco" link to the unit SKU
// of "Refresco em pó Uva".
func newCatalog() *fakeProductCatalog {
	return &fakeProductCatalog{
		products: map[string]pmsDomain.Product{
			"p-case": {ID: "p-case", Name: "Caixa Refresco"},
			"p-unit": {ID: "p-unit", Name: "Refresco em pó Uva - MID - Ajinomoto"},
		},
		variants: map[string]pmsDomain.Variant{
			"v-case": {ID: "v-case", ProductID: "p-case"},
			"v-unit": {ID: "v-unit", ProductID: "p-unit"},
		},
		skus: map[string]pmsDomain.SKU{
			"s-case": {ID: "s-case", VariantID: "v-case", SKUCode: "CX-30", CaseOfSKUID: strp("s-unit"), UnitsPerCase: int64Ptr(30)},
			"s-unit": {ID: "s-unit", VariantID: "v-unit", SKUCode: "REF-UVA", Barcode: "7891132082346"},
		},
	}
}

func newGetProduct(t *testing.T, c *fakeProductCatalog) *apppms.GetProductUseCase {
	return apppms.NewGetProductUseCase(newMockGormDB(t), c, fakeCatalogVariants{c}, fakeCatalogSKUs{c})
}

func TestGetProduct_summarizesLinkedSKUs(t *testing.T) {
	c := newCatalog()

	detail, err := newGetProduct(t, c).Execute(context.Background(), "p-case")

	require.NoError(t, err)
	assert.Equal(t, map[string]apppms.LinkedSKU{
		"s-unit": {ID: "s-unit", ProductName: "Refresco em pó Uva - MID - Ajinomoto", SKUCode: "REF-UVA", Barcode: "7891132082346"},
	}, detail.LinkedSKUs)
}

func TestGetProduct_summarizesPortionedFromSKUs(t *testing.T) {
	c := newCatalog()
	s := c.skus["s-case"]
	s.CaseOfSKUID, s.UnitsPerCase, s.PortionedFromSKUID = nil, nil, strp("s-unit")
	c.skus["s-case"] = s

	detail, err := newGetProduct(t, c).Execute(context.Background(), "p-case")

	require.NoError(t, err)
	assert.Contains(t, detail.LinkedSKUs, "s-unit")
}

func TestGetProduct_noLinksNoSummaries(t *testing.T) {
	detail, err := newGetProduct(t, newCatalog()).Execute(context.Background(), "p-unit")

	require.NoError(t, err)
	assert.Empty(t, detail.LinkedSKUs)
}

func TestGetProduct_aLinkToAMissingSKUIsSkipped(t *testing.T) {
	c := newCatalog()
	delete(c.skus, "s-unit")

	detail, err := newGetProduct(t, c).Execute(context.Background(), "p-case")

	require.NoError(t, err, "a stale link must not break loading the product")
	assert.Empty(t, detail.LinkedSKUs)
}
