package theme

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type CreateSessionInput struct {
	ThemeID string
	UserID  string
	Role    string
	Purpose theme.SessionPurpose
	// Locale is the merchant's language ("" means English).
	Locale locale.Locale
}

// SessionToken is the only time the raw token exists outside the
// caller's browser -- only its hash is stored.
type SessionToken struct {
	Token     string
	ExpiresAt time.Time
}

// CreateSessionUseCase mints an edit session (the admin app hands it to
// the storefront-hosted editor) or a preview session (renders the draft on
// the storefront without touching what shoppers see).
type CreateSessionUseCase struct{ deps Deps }

func NewCreateSessionUseCase(deps Deps) *CreateSessionUseCase {
	return &CreateSessionUseCase{deps: deps}
}

func (uc *CreateSessionUseCase) Execute(ctx context.Context, in CreateSessionInput) (*SessionToken, error) {
	raw, err := uc.deps.NewToken()
	if err != nil {
		return nil, fmt.Errorf("createSession: token: %w", err)
	}
	var out SessionToken
	err = tenant.WithTenant(ctx, uc.deps.DB, func(ctx context.Context) error {
		th, err := uc.deps.Themes.FindByID(ctx, in.ThemeID)
		if err != nil {
			return err
		}
		if _, err := uc.deps.requireThemedChannel(ctx, th.SalesChannelID); err != nil {
			return err
		}
		s, err := theme.NewSession(th.TenantID, th.ID, in.UserID, in.Role, in.Purpose, in.Locale, raw, uc.deps.Now())
		if err != nil {
			return err
		}
		out = SessionToken{Token: raw, ExpiresAt: s.ExpiresAt}
		return uc.deps.Sessions.Create(ctx, &s)
	})
	if err != nil {
		return nil, fmt.Errorf("createSession: %w", err)
	}
	return &out, nil
}
