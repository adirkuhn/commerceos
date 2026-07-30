package state

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// ConnectState is the pending-connect context carried across an external
// OAuth-style redirect (e.g. WooCommerce's Application Authentication flow).
type ConnectState struct {
	TenantID string
	StoreURL string
}

// MintConnectState signs a short-lived {tenantID, storeURL, expiry, nonce}
// token, since the eventual callback has no JWT to read a tenant/store from.
// Not a JWT — a plain HMAC-signed payload is enough here and keeps this
// dependency-free.
func MintConnectState(secret []byte, tenantID, storeURL string, now time.Time, ttl time.Duration) (string, error) {
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	expiresAt := now.Add(ttl).Unix()
	payload := fmt.Sprintf("%s|%d|%s|%s",
		tenantID,
		expiresAt,
		base64.RawURLEncoding.EncodeToString(nonce),
		base64.RawURLEncoding.EncodeToString([]byte(storeURL)),
	)
	mac := sign(secret, payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac), nil
}

// ParseConnectState verifies signature and expiry (relative to now, an
// injectable clock so tests don't need real sleeps) and returns the state.
func ParseConnectState(secret []byte, token string, now time.Time) (ConnectState, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return ConnectState{}, errors.New("malformed state token")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ConnectState{}, errors.New("malformed state token")
	}
	gotMAC, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ConnectState{}, errors.New("malformed state token")
	}

	wantMAC := sign(secret, string(payloadBytes))
	if !hmac.Equal(gotMAC, wantMAC) {
		return ConnectState{}, errors.New("invalid state token signature")
	}

	fields := strings.SplitN(string(payloadBytes), "|", 4)
	if len(fields) != 4 {
		return ConnectState{}, errors.New("malformed state payload")
	}
	tenantID := fields[0]
	expiresAt, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return ConnectState{}, errors.New("malformed state payload")
	}
	if now.Unix() > expiresAt {
		return ConnectState{}, errors.New("state token expired")
	}
	storeURLBytes, err := base64.RawURLEncoding.DecodeString(fields[3])
	if err != nil {
		return ConnectState{}, errors.New("malformed state payload")
	}

	return ConnectState{TenantID: tenantID, StoreURL: string(storeURLBytes)}, nil
}

func sign(secret []byte, payload string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(payload))
	return h.Sum(nil)
}
