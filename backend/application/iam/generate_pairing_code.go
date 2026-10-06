package iam

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"math/big"
	"time"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// pairingCodeTTL is how long a generated code stays scannable -- short
// enough that a code left on a shared screen isn't a standing risk, long
// enough for someone to walk to the handheld and type/scan it.
const pairingCodeTTL = 5 * time.Minute

// maxPairingCodeGenerationAttempts bounds retries when a freshly generated
// code collides with another still-pending code -- collisions are rare (1 in
// up to 1,000,000) so a handful of attempts is already generous.
const maxPairingCodeGenerationAttempts = 5

var errPairingCodeGenerationFailed = errors.New("could not generate a unique pairing code, try again")

type GeneratePairingCodeInput struct {
	// UserID/Role are the desktop caller's own JWT claims -- the scanner
	// that claims this code acts as this exact identity (see
	// ClaimPairingCodeUseCase).
	UserID string
	Role   string
}

type GeneratePairingCodeUseCase struct {
	db     *gorm.DB
	repo   iamDomain.ScannerPairingCodeRepository
	outbox eventrecord.Enqueuer
}

func NewGeneratePairingCodeUseCase(db *gorm.DB, repo iamDomain.ScannerPairingCodeRepository, outbox eventrecord.Enqueuer) *GeneratePairingCodeUseCase {
	return &GeneratePairingCodeUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *GeneratePairingCodeUseCase) Execute(ctx context.Context, in GeneratePairingCodeInput) (*iamDomain.ScannerPairingCode, error) {
	var pairing iamDomain.ScannerPairingCode
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for attempt := 0; attempt < maxPairingCodeGenerationAttempts; attempt++ {
			code, err := randomSixDigitCode()
			if err != nil {
				return err
			}
			existing, err := uc.repo.FindPendingByCode(ctx, code)
			if err != nil {
				return err
			}
			if existing != nil {
				continue
			}
			token, err := randomPairingToken()
			if err != nil {
				return err
			}
			now := time.Now()
			pairing = iamDomain.ScannerPairingCode{
				TenantID: tenant.IDFromContext(ctx), Code: code, Token: token, UserID: in.UserID, Role: in.Role,
				Status: iamDomain.ScannerPairingCodeStatusPending, ExpiresAt: now.Add(pairingCodeTTL), CreatedAt: now,
			}
			if err := uc.repo.Create(ctx, &pairing); err != nil {
				return err
			}
			return eventrecord.Record(ctx, uc.outbox, events.ScannerPairingCodeCreated, pairing.ID, eventrecord.Data{
				"user_id": pairing.UserID, "role": pairing.Role, "expires_at": pairing.ExpiresAt,
			})
		}
		return errPairingCodeGenerationFailed
	})
	if err != nil {
		return nil, fmt.Errorf("generatePairingCode: %w", err)
	}
	return &pairing, nil
}

func randomSixDigitCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("randomSixDigitCode: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// randomPairingToken is the opaque, unguessable counterpart to
// randomSixDigitCode -- 128 bits of entropy, base64url so it round-trips
// cleanly through QR codes, barcodes, and JSON without escaping.
func randomPairingToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("randomPairingToken: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
