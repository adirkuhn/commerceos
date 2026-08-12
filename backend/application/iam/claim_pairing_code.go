package iam

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// scannerSessionTTL is deliberately much longer than pairingCodeTTL -- the
// code itself is a short-lived handoff, but once claimed the scanner should
// stay logged in for a normal warehouse shift without re-pairing.
const scannerSessionTTL = 12 * time.Hour

// claims mirrors middleware.Claims exactly (same field names/JSON tags) so
// the token this use case mints validates through the existing
// middleware.JWT unchanged. Redeclared locally rather than imported: domain/
// and application/ must have zero imports from infrastructure/ (see
// CLAUDE.md's architecture rules), and middleware.Claims lives in
// infrastructure/middleware.
type claims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// ClaimPairingCodeUseCase is called by the public, unauthenticated
// /iam/scanner-pairing/claim endpoint -- the scanner has no tenant identity
// yet, so the code lookup runs under tenant.WithSuperAdmin (bypass_rls),
// same cross-tenant-read pattern cmd/scheduler uses for polling. It mints a
// same-shape JWT using the same JWT_SECRET the JWT middleware already
// validates against, so the returned token needs no new auth path.
type ClaimPairingCodeUseCase struct {
	db        *gorm.DB
	repo      iamDomain.ScannerPairingCodeRepository
	jwtSecret []byte
}

func NewClaimPairingCodeUseCase(db *gorm.DB, repo iamDomain.ScannerPairingCodeRepository, jwtSecret []byte) *ClaimPairingCodeUseCase {
	return &ClaimPairingCodeUseCase{db: db, repo: repo, jwtSecret: jwtSecret}
}

// Execute claims a code via the guessable, rate-limited digit-code path --
// see infrastructure/handlers/scannerauth for where that limiter is applied.
func (uc *ClaimPairingCodeUseCase) Execute(ctx context.Context, code string) (string, error) {
	return uc.claim(ctx, func(ctx context.Context) (*iamDomain.ScannerPairingCode, error) {
		return uc.repo.FindPendingByCode(ctx, code)
	})
}

// ExecuteByToken claims a code via the high-entropy token path (QR/barcode)
// -- unguessable within the pairing TTL, so it carries no rate-limit
// dependency.
func (uc *ClaimPairingCodeUseCase) ExecuteByToken(ctx context.Context, token string) (string, error) {
	return uc.claim(ctx, func(ctx context.Context) (*iamDomain.ScannerPairingCode, error) {
		return uc.repo.FindPendingByToken(ctx, token)
	})
}

func (uc *ClaimPairingCodeUseCase) claim(ctx context.Context, lookup func(context.Context) (*iamDomain.ScannerPairingCode, error)) (string, error) {
	adminCtx := tenant.WithSuperAdmin(ctx)

	var pairing *iamDomain.ScannerPairingCode
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		var err error
		pairing, err = lookup(ctx)
		if err != nil {
			return err
		}
		if pairing == nil {
			return iamDomain.ErrPairingCodeInvalid
		}
		return uc.repo.MarkClaimed(ctx, pairing.ID, time.Now())
	})
	if err != nil {
		return "", fmt.Errorf("claimPairingCode: %w", err)
	}

	now := time.Now()
	c := claims{
		UserID: pairing.UserID, TenantID: pairing.TenantID, Role: pairing.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(scannerSessionTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(uc.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("claimPairingCode: sign token: %w", err)
	}
	return token, nil
}
