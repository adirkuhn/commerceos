package channels

import (
	"context"
	"fmt"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdateSalesChannelInput struct {
	ID   string
	Name string
	// ConnectionID re-targets which integrations.Connection this channel
	// pushes through -- required when the channel's Type needs one (every
	// type implemented today).
	ConnectionID *string
	// EnrollmentMode/EnrollmentCategoryID/EnrollmentTag are how the "add
	// all"/"add by category"/"add by tag" bulk actions work: they're just
	// this channel's default rule changing, evaluated live against the
	// whole catalog with no per-product rows to write.
	EnrollmentMode       channelsDomain.EnrollmentMode
	EnrollmentCategoryID *string
	EnrollmentTag        string
	Active               bool
}

type UpdateSalesChannelUseCase struct {
	db       *gorm.DB
	repo     channelsDomain.SalesChannelRepository
	connRepo integrationsDomain.ConnectionRepository
}

func NewUpdateSalesChannelUseCase(db *gorm.DB, repo channelsDomain.SalesChannelRepository, connRepo integrationsDomain.ConnectionRepository) *UpdateSalesChannelUseCase {
	return &UpdateSalesChannelUseCase{db: db, repo: repo, connRepo: connRepo}
}

func (uc *UpdateSalesChannelUseCase) Execute(ctx context.Context, in UpdateSalesChannelInput) (*channelsDomain.SalesChannel, error) {
	var channel channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		if err := validateSalesChannelInput(existing.Type, in.ConnectionID, in.EnrollmentMode, in.EnrollmentCategoryID, in.EnrollmentTag); err != nil {
			return err
		}
		if in.ConnectionID != nil {
			conn, err := uc.connRepo.FindByID(ctx, *in.ConnectionID)
			if err != nil {
				return err
			}
			if string(conn.Provider) != string(existing.Type) {
				return channelsDomain.ErrChannelTypeConnectionMismatch
			}
		}
		existing.Name = in.Name
		existing.ConnectionID = in.ConnectionID
		existing.EnrollmentMode = in.EnrollmentMode
		existing.EnrollmentCategoryID = in.EnrollmentCategoryID
		existing.EnrollmentTag = in.EnrollmentTag
		existing.Active = in.Active
		existing.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, existing); err != nil {
			return err
		}
		channel = *existing
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("updateSalesChannel: %w", err)
	}
	return &channel, nil
}
