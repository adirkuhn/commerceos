package procurement_test

import (
	"context"
	"errors"
	"testing"

	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCreateProductUC struct {
	out      *pmsDomain.Product
	err      error
	gotInput pmsapp.CreateProductInput
}

func (s *stubCreateProductUC) Execute(_ context.Context, in pmsapp.CreateProductInput) (*pmsDomain.Product, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &pmsDomain.Product{ID: "product-1", Name: in.Name}, nil
}

type stubCreateVariantUC struct {
	out      *pmsDomain.Variant
	err      error
	gotInput pmsapp.CreateVariantInput
}

func (s *stubCreateVariantUC) Execute(_ context.Context, in pmsapp.CreateVariantInput) (*pmsDomain.Variant, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &pmsDomain.Variant{ID: "variant-1", ProductID: in.ProductID, Name: in.Name}, nil
}

type stubCreateSKUUC struct {
	out      *pmsDomain.SKU
	err      error
	gotInput pmsapp.CreateSKUInput
}

func (s *stubCreateSKUUC) Execute(_ context.Context, in pmsapp.CreateSKUInput) (*pmsDomain.SKU, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &pmsDomain.SKU{ID: "sku-1", VariantID: in.VariantID, SKUCode: in.SKUCode}, nil
}

type stubLinkUC struct {
	out      *procurementDomain.PurchaseOrderLineItem
	err      error
	gotInput appprocurement.LinkLineItemToSKUInput
}

func (s *stubLinkUC) Execute(_ context.Context, in appprocurement.LinkLineItemToSKUInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	skuID := in.SKUID
	return &procurementDomain.PurchaseOrderLineItem{ID: in.LineItemID, SKUID: &skuID}, nil
}

func TestCreateProductAndLinkLineItem_composesProductVariantSKUAndLink(t *testing.T) {
	createProduct := &stubCreateProductUC{}
	createVariant := &stubCreateVariantUC{}
	createSKU := &stubCreateSKUUC{}
	link := &stubLinkUC{}
	uc := appprocurement.NewCreateProductAndLinkLineItemUseCase(createProduct, createVariant, createSKU, link)

	result, err := uc.Execute(context.Background(), appprocurement.CreateProductAndLinkLineItemInput{
		LineItemID: "line-item-1", ProductName: "New Widget", SKUCode: "NEW-001",
		Barcode: "1234567890", PriceMinorUnits: 999, CurrencyCode: "USD",
	})

	require.NoError(t, err)
	assert.Equal(t, "New Widget", createProduct.gotInput.Name)
	assert.Equal(t, "product-1", createVariant.gotInput.ProductID)
	assert.Equal(t, "variant-1", createSKU.gotInput.VariantID)
	assert.Equal(t, "NEW-001", createSKU.gotInput.SKUCode)
	assert.Equal(t, "line-item-1", link.gotInput.LineItemID)
	assert.Equal(t, "sku-1", link.gotInput.SKUID)
	require.NotNil(t, result.SKUID)
	assert.Equal(t, "sku-1", *result.SKUID)
}

func TestCreateProductAndLinkLineItem_propagatesCreateProductError(t *testing.T) {
	createProduct := &stubCreateProductUC{err: errors.New("duplicate slug")}
	uc := appprocurement.NewCreateProductAndLinkLineItemUseCase(createProduct, &stubCreateVariantUC{}, &stubCreateSKUUC{}, &stubLinkUC{})

	_, err := uc.Execute(context.Background(), appprocurement.CreateProductAndLinkLineItemInput{LineItemID: "line-item-1", ProductName: "X"})

	require.Error(t, err)
}

func TestCreateProductAndLinkLineItem_propagatesCreateSKUError(t *testing.T) {
	createSKU := &stubCreateSKUUC{err: errors.New("duplicate sku code")}
	uc := appprocurement.NewCreateProductAndLinkLineItemUseCase(&stubCreateProductUC{}, &stubCreateVariantUC{}, createSKU, &stubLinkUC{})

	_, err := uc.Execute(context.Background(), appprocurement.CreateProductAndLinkLineItemInput{LineItemID: "line-item-1", ProductName: "X", SKUCode: "DUP"})

	require.Error(t, err)
}

func TestCreateProductAndLinkLineItem_propagatesLinkError(t *testing.T) {
	link := &stubLinkUC{err: procurementDomain.ErrLineItemNotFound}
	uc := appprocurement.NewCreateProductAndLinkLineItemUseCase(&stubCreateProductUC{}, &stubCreateVariantUC{}, &stubCreateSKUUC{}, link)

	_, err := uc.Execute(context.Background(), appprocurement.CreateProductAndLinkLineItemInput{LineItemID: "missing", ProductName: "X"})

	assert.ErrorIs(t, err, procurementDomain.ErrLineItemNotFound)
}
