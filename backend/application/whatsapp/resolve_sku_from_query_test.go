package whatsapp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type fakeResolveSKURepo struct {
	pmsDomain.SKURepository
	byCode      map[string]pmsDomain.SKU
	byVariantID map[string][]pmsDomain.SKU
}

func (f *fakeResolveSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if s, ok := f.byCode[code]; ok {
		return &s, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}

func (f *fakeResolveSKURepo) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return f.byVariantID[variantID], nil
}

type fakeResolveProductRepo struct {
	pmsDomain.ProductRepository
	byToken map[string][]pmsDomain.Product
}

func (f *fakeResolveProductRepo) SearchByAnyNameToken(_ context.Context, tokens []string, _ int) ([]pmsDomain.Product, error) {
	seen := map[string]pmsDomain.Product{}
	for _, t := range tokens {
		for _, p := range f.byToken[t] {
			seen[p.ID] = p
		}
	}
	out := make([]pmsDomain.Product, 0, len(seen))
	for _, p := range seen {
		out = append(out, p)
	}
	return out, nil
}

type fakeResolveVariantRepo struct {
	pmsDomain.VariantRepository
	byProductID map[string][]pmsDomain.Variant
}

func (f *fakeResolveVariantRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	return f.byProductID[productID], nil
}

func TestResolveSKUFromQuery_exactCodeMatch(t *testing.T) {
	skuRepo := &fakeResolveSKURepo{byCode: map[string]pmsDomain.SKU{"ABC123": {ID: "sku-1", SKUCode: "ABC123"}}}
	uc := appwhatsapp.NewResolveSKUFromQueryUseCase(newMockGormDB(t, 1), skuRepo, &fakeResolveProductRepo{}, &fakeResolveVariantRepo{})

	code, err := uc.Execute(context.Background(), "abc123")

	require.NoError(t, err)
	assert.Equal(t, "ABC123", code, "lowercase typed code still resolves via the uppercased attempt")
}

func TestResolveSKUFromQuery_fuzzyNameMatch_unambiguousSingleSKU(t *testing.T) {
	skuRepo := &fakeResolveSKURepo{
		byVariantID: map[string][]pmsDomain.SKU{"var-1": {{ID: "sku-1", SKUCode: "CHA-LIMAO-1"}}},
	}
	productRepo := &fakeResolveProductRepo{byToken: map[string][]pmsDomain.Product{
		"cha":      {{ID: "prod-1", Name: "Cha de Limão e Gengibre"}},
		"limao":    {{ID: "prod-1", Name: "Cha de Limão e Gengibre"}},
		"gengibre": {{ID: "prod-1", Name: "Cha de Limão e Gengibre"}},
	}}
	variantRepo := &fakeResolveVariantRepo{byProductID: map[string][]pmsDomain.Variant{"prod-1": {{ID: "var-1", ProductID: "prod-1"}}}}
	uc := appwhatsapp.NewResolveSKUFromQueryUseCase(newMockGormDB(t, 1), skuRepo, productRepo, variantRepo)

	code, err := uc.Execute(context.Background(), "tem estoque do cha de limao e gengibre")

	require.NoError(t, err)
	assert.Equal(t, "CHA-LIMAO-1", code)
}

func TestResolveSKUFromQuery_ambiguousNameMatch_returnsEmpty(t *testing.T) {
	productRepo := &fakeResolveProductRepo{byToken: map[string][]pmsDomain.Product{
		"cha": {
			{ID: "prod-1", Name: "Cha Verde"},
			{ID: "prod-2", Name: "Cha Preto"},
		},
	}}
	uc := appwhatsapp.NewResolveSKUFromQueryUseCase(newMockGormDB(t, 1), &fakeResolveSKURepo{}, productRepo, &fakeResolveVariantRepo{})

	code, err := uc.Execute(context.Background(), "cha")

	require.NoError(t, err)
	assert.Empty(t, code, "no clear winner -- both candidates score identically low, must not guess")
}

func TestResolveSKUFromQuery_nameMatchesMultiVariantProduct_returnsEmpty(t *testing.T) {
	skuRepo := &fakeResolveSKURepo{byVariantID: map[string][]pmsDomain.SKU{
		"var-1": {{ID: "sku-1", SKUCode: "SHIRT-S"}},
		"var-2": {{ID: "sku-2", SKUCode: "SHIRT-M"}},
	}}
	productRepo := &fakeResolveProductRepo{byToken: map[string][]pmsDomain.Product{
		"camiseta": {{ID: "prod-1", Name: "Camiseta Brasil"}},
		"brasil":   {{ID: "prod-1", Name: "Camiseta Brasil"}},
	}}
	variantRepo := &fakeResolveVariantRepo{byProductID: map[string][]pmsDomain.Variant{
		"prod-1": {{ID: "var-1", ProductID: "prod-1"}, {ID: "var-2", ProductID: "prod-1"}},
	}}
	uc := appwhatsapp.NewResolveSKUFromQueryUseCase(newMockGormDB(t, 1), skuRepo, productRepo, variantRepo)

	code, err := uc.Execute(context.Background(), "voce tem camiseta brasil")

	require.NoError(t, err)
	assert.Empty(t, code, "multi-variant product can't be safely narrowed to one SKU by name text alone")
}

func TestResolveSKUFromQuery_noCandidates_returnsEmpty(t *testing.T) {
	uc := appwhatsapp.NewResolveSKUFromQueryUseCase(newMockGormDB(t, 1), &fakeResolveSKURepo{}, &fakeResolveProductRepo{}, &fakeResolveVariantRepo{})

	code, err := uc.Execute(context.Background(), "xyzzyunknownthing")

	require.NoError(t, err)
	assert.Empty(t, code)
}

func TestResolveSKUFromQuery_emptyQuery_returnsEmptyWithoutOpeningATransaction(t *testing.T) {
	uc := appwhatsapp.NewResolveSKUFromQueryUseCase(newMockGormDB(t, 0), &fakeResolveSKURepo{}, &fakeResolveProductRepo{}, &fakeResolveVariantRepo{})

	code, err := uc.Execute(context.Background(), "   ")

	require.NoError(t, err)
	assert.Empty(t, code)
}
