package lm2license

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidLicense = errors.New("invalid license")

// PublicKeyFetcher fetches lm2's current RSA signing public key (PEM).
// HTTPPublicKeyFetcher is the production implementation.
type PublicKeyFetcher interface {
	GetPublicKey() (string, error)
}

// TokenService verifies an lm2-issued license's signature and extracts its
// payload. Ported from the POS desktop's own license-verification code
// (same lm2-issued token format) -- see IsUsable's doc comment for the one
// deliberate behavior difference from that original.
type TokenService struct {
	fetcher PublicKeyFetcher
}

func NewTokenService(fetcher PublicKeyFetcher) *TokenService {
	return &TokenService{fetcher: fetcher}
}

// VerifyAndGetPayload verifies licenseKey's RS256 signature against lm2's
// current public key and returns its payload. Signature-valid only --
// callers MUST additionally check payload.IsUsable(time.Now()) before
// treating the caller as authenticated; this deliberately does not enforce
// expiry/status itself so a stale cached public key can't accidentally
// mask a real rejection reason.
func (h *TokenService) VerifyAndGetPayload(licenseKey string) (*LicensePayload, error) {
	pubKeyPEM, err := h.fetcher.GetPublicKey()
	if err != nil {
		return nil, fmt.Errorf("fetch lm2 public key: %w", err)
	}

	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("invalid lm2 public key PEM")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse lm2 public key: %w", err)
	}

	// Skip JWT standard claims (exp/nbf/iat) -- the license payload has its
	// own expires_at/status that the caller checks via IsUsable. Skipping
	// here means a short-lived JWT wrapper won't break license checks.
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.Parse(licenseKey, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return pubKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLicense, err)
	}
	if !token.Valid {
		return nil, ErrInvalidLicense
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidLicense
	}
	payloadStr, ok := claims["payload"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing payload claim", ErrInvalidLicense)
	}

	var payload LicensePayload
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidLicense, err)
	}
	return &payload, nil
}
