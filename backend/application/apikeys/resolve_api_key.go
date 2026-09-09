package apikeys

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// internalChannelFinder is channels.SalesChannelRepository narrowed to
// point of use (ISP) -- resolving a channel-id-as-bearer-token only ever
// needs a lookup by id.
type internalChannelFinder interface {
	FindByID(ctx context.Context, id string) (*channelsDomain.SalesChannel, error)
}

// ResolveAPIKeyUseCase turns a storefront's bearer key into the tenant it
// belongs to. The caller (infrastructure/handlers/storefront) has no
// tenant identity yet -- that's exactly what a successful resolution
// establishes -- so the lookup runs under tenant.WithSuperAdmin (RLS
// bypass), the same cross-tenant-read-by-secret pattern
// application/iam.ClaimPairingCodeUseCase uses to resolve a scanner
// pairing code before any tenant/session exists.
type ResolveAPIKeyUseCase struct {
	db          *gorm.DB
	repo        apikeysDomain.Repository
	channelRepo internalChannelFinder
}

func NewResolveAPIKeyUseCase(db *gorm.DB, repo apikeysDomain.Repository, channelRepo internalChannelFinder) *ResolveAPIKeyUseCase {
	return &ResolveAPIKeyUseCase{db: db, repo: repo, channelRepo: channelRepo}
}

// Execute returns the ResolvedAPIKey (tenant, and sales channel if the key
// is scoped) the plaintext key belongs to, or apikeysDomain.ErrNotFound if
// it matches no active key -- the handler maps that to a 401, never
// leaking whether the key was wrong or revoked.
//
// A bearer value that parses as a UUID and isn't a known hashed key is
// tried a second way: as an internal SalesChannel's own id. That channel's
// own id is what the admin app shows as that channel's storefront URL (see
// frontend/src/pages/SalesChannels.tsx) -- no separate generated secret to
// manage, safe to reuse this way because a resolved internal channel only
// ever grants the same public/anonymous-shopper privilege level a
// dedicated storefront key would (see ResolvedAPIKey's doc comment on the
// Stripe-publishable-key trust model this already assumes).
func (uc *ResolveAPIKeyUseCase) Execute(ctx context.Context, plaintextKey string) (apikeysDomain.ResolvedAPIKey, error) {
	if plaintextKey == "" {
		return apikeysDomain.ResolvedAPIKey{}, apikeysDomain.ErrNotFound
	}

	adminCtx := tenant.WithSuperAdmin(ctx)
	var resolved apikeysDomain.ResolvedAPIKey
	err := tenant.WithTenant(adminCtx, uc.db, func(ctx context.Context) error {
		found, err := uc.repo.FindActiveTenantIDByHash(ctx, hashKey(plaintextKey))
		if err == nil {
			resolved = found
			return nil
		}
		if !errors.Is(err, apikeysDomain.ErrNotFound) || uuid.Validate(plaintextKey) != nil {
			return err
		}
		channel, chErr := uc.channelRepo.FindByID(ctx, plaintextKey)
		if chErr != nil || channel.Type != channelsDomain.ChannelTypeInternal {
			return err
		}
		resolved = apikeysDomain.ResolvedAPIKey{TenantID: channel.TenantID, SalesChannelID: &channel.ID}
		return nil
	})
	if err != nil {
		return apikeysDomain.ResolvedAPIKey{}, fmt.Errorf("resolveApiKey: %w", err)
	}
	return resolved, nil
}
