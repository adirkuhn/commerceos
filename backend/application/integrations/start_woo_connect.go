package integrations

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/internal/state"
	"github.com/adirkuhn/commerceos/internal/tenant"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

const connectStateTTL = 10 * time.Minute

type StartWooConnectInput struct {
	StoreURL string
}

type StartWooConnectResult struct {
	AuthorizeURL string
}

// StartWooConnectUseCase builds the WooCommerce "Application Authentication"
// authorize URL. It has no repository dependency — it reads the tenant from
// the authenticated request's context and mints a signed state token that
// carries that tenant (+ store URL) across the redirect to the unauthenticated
// callback, which has no JWT to read a tenant from.
type StartWooConnectUseCase struct {
	stateSecret      []byte
	publicAPIBaseURL string
	frontendBaseURL  string
}

func NewStartWooConnectUseCase(stateSecret []byte, publicAPIBaseURL, frontendBaseURL string) *StartWooConnectUseCase {
	return &StartWooConnectUseCase{
		stateSecret:      stateSecret,
		publicAPIBaseURL: publicAPIBaseURL,
		frontendBaseURL:  frontendBaseURL,
	}
}

func (uc *StartWooConnectUseCase) Execute(ctx context.Context, in StartWooConnectInput) (*StartWooConnectResult, error) {
	storeURL := integrationsDomain.StoreURL(in.StoreURL)
	if err := storeURL.Validate(); err != nil {
		return nil, fmt.Errorf("startWooConnect: %w", err)
	}

	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("startWooConnect: %w", integrationsDomain.ErrNoTenantContext)
	}

	token, err := state.MintConnectState(uc.stateSecret, tenantID, in.StoreURL, time.Now(), connectStateTTL)
	if err != nil {
		return nil, fmt.Errorf("startWooConnect: %w", err)
	}

	callbackURL := uc.publicAPIBaseURL + "/api/v1/integrations/woocommerce/connect/callback"
	returnURL := uc.frontendBaseURL + "/integrations"

	authorizeURL := fmt.Sprintf("%s/wc-auth/v1/authorize?%s",
		strings.TrimRight(in.StoreURL, "/"),
		url.Values{
			"app_name":     {"CommerceOS"},
			"scope":        {"read_write"},
			"user_id":      {token},
			"return_url":   {returnURL},
			"callback_url": {callbackURL},
		}.Encode(),
	)

	return &StartWooConnectResult{AuthorizeURL: authorizeURL}, nil
}
