package lm2license

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// publicKeyCacheTTL bounds how long a fetched key is trusted before
// re-fetching -- long enough that a POS hitting the API repeatedly doesn't
// hammer lm2 on every request, short enough that a key rotation on lm2's
// side is picked up within a bounded window rather than requiring a
// CommerceOS restart.
const publicKeyCacheTTL = 15 * time.Minute

// HTTPPublicKeyFetcher fetches lm2's current signing public key over HTTP
// and caches it. baseURL/response shape here are a best-effort guess
// (either a bare PEM body, or {"public_key": "..."} JSON) -- confirm
// against lm2's actual public-key endpoint and adjust before relying on
// this in production; see LM2_LICENSE_PUBLIC_KEY_URL in cmd/api.
type HTTPPublicKeyFetcher struct {
	url        string
	httpClient *http.Client

	mu       sync.RWMutex
	cached   string
	cachedAt time.Time
}

func NewHTTPPublicKeyFetcher(publicKeyURL string) *HTTPPublicKeyFetcher {
	client := &http.Client{Timeout: 5 * time.Second}

	// A .dev.localhost lm2 instance's cert is signed by a local CA nothing
	// but the developer's own machine trusts — same carve-out the POS
	// desktop's own license.Client already applies for this exact host
	// suffix (see akantio_pos/src/license/client.go). A real
	// LM2_LICENSE_PUBLIC_KEY_URL is always a public domain in production,
	// so this never weakens a real deployment's TLS verification.
	if u, err := url.Parse(publicKeyURL); err == nil && strings.HasSuffix(u.Hostname(), ".dev.localhost") {
		client.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}

	return &HTTPPublicKeyFetcher{url: publicKeyURL, httpClient: client}
}

func (f *HTTPPublicKeyFetcher) GetPublicKey() (string, error) {
	f.mu.RLock()
	if f.cached != "" && time.Since(f.cachedAt) < publicKeyCacheTTL {
		key := f.cached
		f.mu.RUnlock()
		return key, nil
	}
	f.mu.RUnlock()

	resp, err := f.httpClient.Get(f.url)
	if err != nil {
		return "", fmt.Errorf("lm2license: fetch public key: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lm2license: fetch public key: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("lm2license: read public key response: %w", err)
	}

	key, err := extractPublicKey(body)
	if err != nil {
		return "", err
	}

	f.mu.Lock()
	f.cached = key
	f.cachedAt = time.Now()
	f.mu.Unlock()
	return key, nil
}

// extractPublicKey accepts lm2's actual response shape for
// GET public-keys/current — {"data": {"public_key": "..."}} — confirmed
// against the POS desktop's own already-working
// license.Client.GetPublicKey (akantio_pos/src/license/client.go). A bare
// PEM body or a flat {"public_key": "..."} without the data wrapper are
// also accepted as harmless fallbacks, in case a future endpoint ever
// responds differently.
func extractPublicKey(body []byte) (string, error) {
	trimmed := strings.TrimSpace(string(body))
	if strings.HasPrefix(trimmed, "-----BEGIN") {
		return trimmed, nil
	}

	var enveloped struct {
		Data struct {
			PublicKey string `json:"public_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &enveloped); err == nil && enveloped.Data.PublicKey != "" {
		return enveloped.Data.PublicKey, nil
	}

	var flat struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.Unmarshal(body, &flat); err == nil && flat.PublicKey != "" {
		return flat.PublicKey, nil
	}

	return "", fmt.Errorf("lm2license: unrecognized public key response format")
}
