package theme

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// editSessionTTL is generous enough to cover one realistic editing visit
// (open the editor, make several rounds of changes, save) without staying
// valid indefinitely -- a leaked/logged token stops being usable well
// within the same day. A merchant whose session expires mid-edit just
// re-opens "Edit visually" from the admin app for a fresh one; no
// in-progress work is lost since Puck saves happen explicitly, not on a
// timer.
const editSessionTTL = 60 * time.Minute

// CreateEditSessionUseCase mints a short-lived EditSession token for one
// internal sales channel -- called from the JWT-protected admin Theme
// Builder API only (see infrastructure/handlers/theme), never from the
// public storefront API. The admin app opens storefront-frontend's
// embedded Puck editor with this token in the URL.
type CreateEditSessionUseCase struct {
	db          *gorm.DB
	repo        themeDomain.EditSessionRepository
	channelRepo channelsDomain.SalesChannelRepository
}

func NewCreateEditSessionUseCase(db *gorm.DB, repo themeDomain.EditSessionRepository, channelRepo channelsDomain.SalesChannelRepository) *CreateEditSessionUseCase {
	return &CreateEditSessionUseCase{db: db, repo: repo, channelRepo: channelRepo}
}

func (uc *CreateEditSessionUseCase) Execute(ctx context.Context, salesChannelID string) (*themeDomain.EditSession, error) {
	var session themeDomain.EditSession
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.channelRepo.FindByID(ctx, salesChannelID); err != nil {
			return err
		}
		now := time.Now()
		session = themeDomain.EditSession{
			TenantID: tenant.IDFromContext(ctx), SalesChannelID: salesChannelID,
			Token: idgen.NewV4(), ExpiresAt: now.Add(editSessionTTL), CreatedAt: now,
		}
		return uc.repo.Create(ctx, &session)
	})
	if err != nil {
		return nil, fmt.Errorf("createEditSession: %w", err)
	}
	return &session, nil
}
