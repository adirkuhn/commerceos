package iam

import "fmt"

// RefreshScannerSessionUseCase reissues the caller's own scanner-session
// token with a fresh scannerSessionTTL expiry, same identity claims. Unlike
// ClaimPairingCodeUseCase (public, no identity yet), this always runs behind
// middleware.JWT -- the caller already proved who they are, so there's no DB
// lookup here, just a re-mint off the claims the handler already validated.
type RefreshScannerSessionUseCase struct {
	jwtSecret []byte
}

func NewRefreshScannerSessionUseCase(jwtSecret []byte) *RefreshScannerSessionUseCase {
	return &RefreshScannerSessionUseCase{jwtSecret: jwtSecret}
}

func (uc *RefreshScannerSessionUseCase) Execute(userID, tenantID, role string) (string, error) {
	token, err := mintScannerToken(userID, tenantID, role, uc.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("refreshScannerSession: %w", err)
	}
	return token, nil
}
