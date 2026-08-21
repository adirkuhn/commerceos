package pms_test

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// stubOutboxRepo is a no-op outbox — the tests that need this package's
// UpdateProductUseCase/UpdateSKUUseCase to raise a pms.product.changed
// event only care that Enqueue was called (via `calls`), not that it
// actually persists anything.
type stubOutboxRepo struct {
	calls    []string // subjects enqueued
	payloads [][]byte // one per call, same order as calls
	err      error
}

func (s *stubOutboxRepo) Enqueue(_ context.Context, subject, _ string, payload []byte, _ time.Time) error {
	s.calls = append(s.calls, subject)
	s.payloads = append(s.payloads, payload)
	return s.err
}
func (s *stubOutboxRepo) FetchPending(_ context.Context, _ int) ([]outbox.Message, error) {
	return nil, nil
}
func (s *stubOutboxRepo) MarkPublished(_ context.Context, _ uint) error { return nil }
func (s *stubOutboxRepo) MarkFailed(_ context.Context, _ uint) error    { return nil }

type stubSKURepo struct {
	findByCodeResult     *pmsDomain.SKU
	findByCodeErr        error
	findByIDResult       *pmsDomain.SKU
	findByIDErr          error
	findByBarcodeResult  *pmsDomain.SKU
	findByBarcodeErr     error
	created              *pmsDomain.SKU
	createErr            error
	updated              *pmsDomain.SKU
	gotCode              string
	gotBarcode           string
	listActiveResult     []pmsDomain.SKU
	listActiveErr        error
	updateCostPriceCalls []struct {
		SKUID               string
		CostPriceMinorUnits int64
	}
	updateCostPriceErr      error
	setManualCostPriceCalls []struct {
		SKUID               string
		CostPriceMinorUnits int64
	}
	setManualCostPriceErr error
	clearOverrideCalls    []string
	clearOverrideErr      error
}

func (s *stubSKURepo) Create(_ context.Context, sku *pmsDomain.SKU) error {
	if s.createErr != nil {
		return s.createErr
	}
	sku.ID = "sku-1"
	s.created = sku
	return nil
}
func (s *stubSKURepo) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return &pmsDomain.SKU{ID: "sku-1"}, nil
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	s.gotCode = code
	if s.findByCodeResult == nil && s.findByCodeErr == nil {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return s.findByCodeResult, s.findByCodeErr
}
func (s *stubSKURepo) FindByBarcode(_ context.Context, barcode string) (*pmsDomain.SKU, error) {
	s.gotBarcode = barcode
	if s.findByBarcodeResult == nil && s.findByBarcodeErr == nil {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return s.findByBarcodeResult, s.findByBarcodeErr
}
func (s *stubSKURepo) ListByVariant(_ context.Context, _ string) ([]pmsDomain.SKU, error) {
	return nil, nil
}
func (s *stubSKURepo) Update(_ context.Context, sku *pmsDomain.SKU) error {
	s.updated = sku
	return nil
}
func (s *stubSKURepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubSKURepo) ListWithExpiredSale(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *stubSKURepo) ExistsWithTaxClass(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (s *stubSKURepo) ListActive(_ context.Context) ([]pmsDomain.SKU, error) {
	return s.listActiveResult, s.listActiveErr
}

func (s *stubSKURepo) UpdateCostPrice(_ context.Context, skuID string, costPriceMinorUnits int64) error {
	s.updateCostPriceCalls = append(s.updateCostPriceCalls, struct {
		SKUID               string
		CostPriceMinorUnits int64
	}{skuID, costPriceMinorUnits})
	return s.updateCostPriceErr
}

func (s *stubSKURepo) SetManualCostPrice(_ context.Context, skuID string, costPriceMinorUnits int64) error {
	s.setManualCostPriceCalls = append(s.setManualCostPriceCalls, struct {
		SKUID               string
		CostPriceMinorUnits int64
	}{skuID, costPriceMinorUnits})
	return s.setManualCostPriceErr
}

func (s *stubSKURepo) ClearCostPriceOverride(_ context.Context, skuID string) error {
	s.clearOverrideCalls = append(s.clearOverrideCalls, skuID)
	return s.clearOverrideErr
}
