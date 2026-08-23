package webhookproviders_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/webhookproviders"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestWooCommerceVerifier_acceptsValidSignature(t *testing.T) {
	v := webhookproviders.NewWooCommerceVerifier()
	body := []byte(`{"id":1}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": sign("secret", body)}

	assert.NoError(t, v.Verify(body, headers, "secret"))
}

func TestWooCommerceVerifier_rejectsWrongSecret(t *testing.T) {
	v := webhookproviders.NewWooCommerceVerifier()
	body := []byte(`{"id":1}`)
	headers := map[string]string{"X-Wc-Webhook-Signature": sign("wrong", body)}

	assert.ErrorIs(t, v.Verify(body, headers, "secret"), webhooks.ErrInvalidSignature)
}

func TestWooCommerceVerifier_rejectsTamperedBody(t *testing.T) {
	v := webhookproviders.NewWooCommerceVerifier()
	headers := map[string]string{"X-Wc-Webhook-Signature": sign("secret", []byte(`{"id":1}`))}

	assert.ErrorIs(t, v.Verify([]byte(`{"id":2}`), headers, "secret"), webhooks.ErrInvalidSignature)
}

func TestWooCommerceVerifier_rejectsMissingHeader(t *testing.T) {
	v := webhookproviders.NewWooCommerceVerifier()

	assert.ErrorIs(t, v.Verify([]byte(`{}`), map[string]string{}, "secret"), webhooks.ErrInvalidSignature)
}

func TestStubVerifier_alwaysNotImplemented(t *testing.T) {
	v := webhookproviders.NewStubVerifier()

	assert.ErrorIs(t, v.Verify([]byte(`{}`), map[string]string{}, "any-secret"), webhooks.ErrProviderNotImplemented)
}

func TestNewRegistry_containsWooCommerceAndAllStubProviders(t *testing.T) {
	reg := webhookproviders.NewRegistry()

	for _, name := range []string{"woocommerce", "bol", "moneybird", "shopify", "lightspeed", "myparcel", "sendcloud"} {
		if _, ok := reg[name]; !ok {
			t.Errorf("expected registry to contain provider %q", name)
		}
	}

	// woocommerce and myparcel are real verifiers, not stubs -- the rest
	// still return ErrProviderNotImplemented until built.
	for _, name := range []string{"bol", "moneybird", "shopify", "lightspeed", "sendcloud"} {
		if err := reg[name].Verify(nil, nil, "s"); err == nil {
			t.Errorf("expected provider %q to still be a stub (unimplemented)", name)
		}
	}
	if _, ok := reg["myparcel"].(*webhookproviders.MyParcelVerifier); !ok {
		t.Errorf("expected myparcel to resolve to the real MyParcelVerifier, not a stub")
	}
}
