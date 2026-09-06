package webhookproviders

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/domain/webhooks"
)

// ProviderRevolut is the registry key + URL path segment for Revolut.
const ProviderRevolut = "revolut"

const (
	revolutSignatureHeader = "Revolut-Signature"
	revolutTimestampHeader = "Revolut-Request-Timestamp"
)

// revolutTimestampTolerance mirrors Stripe's own default (5min) -- Revolut's
// docs don't state one explicitly, but the timestamp is only useful against
// replay if some tolerance is enforced, and 5min is this codebase's existing
// convention (see stripe_verifier.go's doc comment).
const revolutTimestampTolerance = 5 * time.Minute

// RevolutVerifier implements domain/webhooks.ProviderVerifier for Revolut's
// Merchant API webhooks. Revolut signs each delivery with HMAC-SHA256:
// header Revolut-Signature carries one or more comma-separated "v1=<hex>"
// values (multiple during secret rotation -- any match is accepted), and
// Revolut-Request-Timestamp carries the signing timestamp in epoch
// milliseconds. The signed string is "v1.{timestamp}.{rawBody}"
// (period-separated, using the exact raw request body).
//
// UNCONFIRMED against a real Revolut account -- built from published docs
// only (developer.revolut.com's webhook-signature guide blocks automated
// fetching), same caveat convention as this codebase's other from-docs-only
// vendor integrations (infrastructure/myparcel, infrastructure/revolut's
// own outbound client). Confirm against a real webhook delivery before
// relying on this in production.
type RevolutVerifier struct{}

func NewRevolutVerifier() *RevolutVerifier { return &RevolutVerifier{} }

func (RevolutVerifier) Verify(rawBody []byte, headers map[string]string, secret string) error {
	sigHeader := headers[revolutSignatureHeader]
	tsHeader := headers[revolutTimestampHeader]
	if sigHeader == "" || tsHeader == "" || secret == "" {
		return webhooks.ErrInvalidSignature
	}

	tsMillis, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return webhooks.ErrInvalidSignature
	}
	ts := time.UnixMilli(tsMillis)
	age := time.Since(ts)
	if age < -revolutTimestampTolerance || age > revolutTimestampTolerance {
		return webhooks.ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v1." + tsHeader + "."))
	mac.Write(rawBody)
	want := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range strings.Split(sigHeader, ",") {
		sig = strings.TrimSpace(sig)
		rest, ok := strings.CutPrefix(sig, "v1=")
		if !ok {
			continue
		}
		// hmac.Equal is constant-time -- same reasoning as
		// WooCommerceVerifier's use of it.
		if hmac.Equal([]byte(rest), []byte(want)) {
			return nil
		}
	}
	return webhooks.ErrInvalidSignature
}

// EventID always defers to ReceiveWebhookUseCase's raw-body-hash fallback.
// Revolut's ORDER_COMPLETED payload has no documented per-delivery id
// (order_id identifies the order, not a specific delivery attempt) -- same
// reasoning as MyParcelVerifier/WooCommerceVerifier.
func (RevolutVerifier) EventID(headers map[string]string, rawBody []byte) string {
	return ""
}
