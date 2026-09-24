package oms

import (
	"context"
	"fmt"
	"strings"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ResolvePackScanCodeUseCase is packing's resolve-scan: turns a scanned code
// into a SKU (+ weight for a scale barcode) without recording anything --
// the packing screen uses it to identify a substitute or extra item before
// staging it. Same resolution chain as picking (scanCodeResolver).
type ResolvePackScanCodeUseCase struct {
	db          *gorm.DB
	sessionRepo omsDomain.PackingSessionRepository
	resolver    scanCodeResolver
}

func NewResolvePackScanCodeUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	skuRepo skuResolverRepo,
	barcodeAliasRepo barcodeAliasFinder,
	scaleCodeAliasRepo scaleCodeAliasFinder,
	variantRepo variantFinderForPickScanResolve,
	productRepo productFinderForPickScanResolve,
) *ResolvePackScanCodeUseCase {
	return &ResolvePackScanCodeUseCase{db: db, sessionRepo: sessionRepo, resolver: scanCodeResolver{
		lineItemRepo: lineItemRepo, skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo,
		scaleCodeAliasRepo: scaleCodeAliasRepo, variantRepo: variantRepo, productRepo: productRepo,
	}}
}

func (uc *ResolvePackScanCodeUseCase) Execute(ctx context.Context, sessionID, code string) (*ResolvedScanCode, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("resolvePackScanCode: %w", omsDomain.ErrPackScanCodeOrSKURequired)
	}

	var result *ResolvedScanCode
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PackingSessionStatusInProgress {
			return omsDomain.ErrPackingSessionNotActive
		}
		result, err = uc.resolver.resolve(ctx, session.SalesOrderID, code)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("resolvePackScanCode: %w", err)
	}
	return result, nil
}
