package pms_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubPreviewProductMerge struct {
	out *apppms.ProductMergePreview
	err error
	got apppms.PreviewProductMergeInput
}

func (s *stubPreviewProductMerge) Execute(_ context.Context, in apppms.PreviewProductMergeInput) (*apppms.ProductMergePreview, error) {
	s.got = in
	return s.out, s.err
}

type stubMergeProducts struct {
	out *apppms.MergeProductsResult
	err error
	got apppms.MergeProductsInput
}

func (s *stubMergeProducts) Execute(_ context.Context, in apppms.MergeProductsInput) (*apppms.MergeProductsResult, error) {
	s.got = in
	return s.out, s.err
}

func TestPreviewProductMerge_success(t *testing.T) {
	c := int64(140)
	s := &stubs{previewProductMerge: stubPreviewProductMerge{out: &apppms.ProductMergePreview{
		Source:  apppms.MergePreviewProduct{ID: "dup", Name: "Dup", SKUs: []apppms.MergePreviewSKU{{ID: "d1", SKUCode: "D-1", OnHand: 4}}},
		Target:  apppms.MergePreviewProduct{ID: "orig", Name: "Orig", SKUs: []apppms.MergePreviewSKU{{ID: "o1", SKUCode: "O-1", OnHand: 6}}},
		Pairs:   []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o1"}},
		Targets: []apppms.MergePreviewTarget{{SKUID: "o1", OnHandAfter: 10, CostAfter: &c}},
	}}}

	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/dup/merge-preview", map[string]any{"target_product_id": "orig"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, apppms.PreviewProductMergeInput{SourceProductID: "dup", TargetProductID: "orig"}, s.previewProductMerge.got)
	body := w.Body.String()
	assert.Contains(t, body, `"sku_mapping":[{"source_sku_id":"d1","target_sku_id":"o1"}]`)
	assert.Contains(t, body, `"cost_after_minor_units":140`)
	assert.Contains(t, body, `"on_hand_after":10`)
}

func TestPreviewProductMerge_passesTheGivenMapping(t *testing.T) {
	s := &stubs{previewProductMerge: stubPreviewProductMerge{out: &apppms.ProductMergePreview{}}}

	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/dup/merge-preview", map[string]any{
		"target_product_id": "orig", "sku_mapping": []map[string]string{{"source_sku_id": "d1", "target_sku_id": "o2"}},
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []pmsDomain.SKUMergePair{{SourceSKUID: "d1", TargetSKUID: "o2"}}, s.previewProductMerge.got.Pairs)
}

func TestPreviewProductMerge_missingTargetIsUnprocessable(t *testing.T) {
	s := &stubs{}
	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/dup/merge-preview", map[string]any{})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestMergeProduct_success(t *testing.T) {
	s := &stubs{mergeProducts: stubMergeProducts{out: &apppms.MergeProductsResult{
		TargetProductID: "orig", SKUs: []pmsDomain.MergedSKU{{SourceSKUID: "d1", SourceSKUCode: "D-1", SourceBarcode: "871", TargetSKUID: "o1"}},
	}}}

	w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/dup/merge", map[string]any{"target_product_id": "orig"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "dup", s.mergeProducts.got.SourceProductID)
	assert.Equal(t, "orig", s.mergeProducts.got.TargetProductID)
	assert.Contains(t, w.Body.String(), `"target_product_id":"orig"`)
	assert.Contains(t, w.Body.String(), `"source_sku_code":"D-1"`)
}

func TestMergeProduct_errors(t *testing.T) {
	for name, c := range map[string]struct {
		err  error
		want int
	}{
		"not an admin":       {pmsDomain.ErrProductMergeForbidden, http.StatusForbidden},
		"unknown product":    {pmsDomain.ErrProductNotFound, http.StatusNotFound},
		"incomplete mapping": {pmsDomain.ErrMergeMappingIncomplete, http.StatusUnprocessableEntity},
		"merge into itself":  {pmsDomain.ErrMergeIntoSelf, http.StatusUnprocessableEntity},
	} {
		t.Run(name, func(t *testing.T) {
			s := &stubs{mergeProducts: stubMergeProducts{err: c.err}}
			w := doRequest(newTestRouter(s), http.MethodPost, "/api/v1/pms/products/dup/merge", map[string]any{"target_product_id": "orig"})
			assert.Equal(t, c.want, w.Code, w.Body.String())
		})
	}
}
