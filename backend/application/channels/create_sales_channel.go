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

type CreateSalesChannelInput struct {
	Name                       string
	Type                       channelsDomain.ChannelType
	ConnectionID               *string
	EnrollmentMode             channelsDomain.EnrollmentMode
	EnrollmentCategoryID       *string
	EnrollmentTag              string
	PriceAdjustmentBasisPoints *int
	QuantityBreaks             []channelsDomain.QuantityBreak
}

// validateSalesChannelInput is shared by Create and Update -- same field
// rules apply whichever way a channel got into this shape.
func validateSalesChannelInput(channelType channelsDomain.ChannelType, connectionID *string, mode channelsDomain.EnrollmentMode, categoryID *string, tag string, priceAdjustmentBasisPoints *int, quantityBreaks []channelsDomain.QuantityBreak) error {
	if !channelType.Valid() {
		return channelsDomain.ErrInvalidChannelType
	}
	if !mode.Valid() {
		return channelsDomain.ErrInvalidEnrollmentMode
	}
	if channelType.RequiresConnection() && (connectionID == nil || *connectionID == "") {
		return channelsDomain.ErrConnectionRequired
	}
	if mode == channelsDomain.EnrollmentModeCategory && (categoryID == nil || *categoryID == "") {
		return channelsDomain.ErrEnrollmentCategoryRequired
	}
	if mode == channelsDomain.EnrollmentModeTag && tag == "" {
		return channelsDomain.ErrEnrollmentTagRequired
	}
	if priceAdjustmentBasisPoints != nil && (*priceAdjustmentBasisPoints < 0 || *priceAdjustmentBasisPoints > 10000) {
		return channelsDomain.ErrInvalidPriceAdjustment
	}
	seenMinQuantity := map[int64]bool{}
	for _, b := range quantityBreaks {
		if b.MinQuantity <= 0 || b.DiscountBasisPoints < 0 || b.DiscountBasisPoints > 10000 || seenMinQuantity[b.MinQuantity] {
			return channelsDomain.ErrInvalidQuantityBreak
		}
		seenMinQuantity[b.MinQuantity] = true
	}
	return nil
}

type CreateSalesChannelUseCase struct {
	db       *gorm.DB
	repo     channelsDomain.SalesChannelRepository
	connRepo integrationsDomain.ConnectionRepository
}

func NewCreateSalesChannelUseCase(db *gorm.DB, repo channelsDomain.SalesChannelRepository, connRepo integrationsDomain.ConnectionRepository) *CreateSalesChannelUseCase {
	return &CreateSalesChannelUseCase{db: db, repo: repo, connRepo: connRepo}
}

func (uc *CreateSalesChannelUseCase) Execute(ctx context.Context, in CreateSalesChannelInput) (*channelsDomain.SalesChannel, error) {
	if err := validateSalesChannelInput(in.Type, in.ConnectionID, in.EnrollmentMode, in.EnrollmentCategoryID, in.EnrollmentTag, in.PriceAdjustmentBasisPoints, in.QuantityBreaks); err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}

	var channel channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if in.ConnectionID != nil {
			conn, err := uc.connRepo.FindByID(ctx, *in.ConnectionID)
			if err != nil {
				return err
			}
			if string(conn.Provider) != string(in.Type) {
				return channelsDomain.ErrChannelTypeConnectionMismatch
			}
		}
		now := time.Now()
		channel = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: in.Name, Type: in.Type, ConnectionID: in.ConnectionID,
			EnrollmentMode: in.EnrollmentMode, EnrollmentCategoryID: in.EnrollmentCategoryID, EnrollmentTag: in.EnrollmentTag,
			PriceAdjustmentBasisPoints: in.PriceAdjustmentBasisPoints,
			QuantityBreaks:             in.QuantityBreaks,
			Active:                     true, CreatedAt: now, UpdatedAt: now,
		}
		return uc.repo.Create(ctx, &channel)
	})
	if err != nil {
		return nil, fmt.Errorf("createSalesChannel: %w", err)
	}
	return &channel, nil
}
