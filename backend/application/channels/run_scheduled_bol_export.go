package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// runScheduledBolExportLimit mirrors runScheduledProductExportLimit's
// reasoning -- a fixed, generous page size rather than real pagination.
const runScheduledBolExportLimit = 200

type RunScheduledBolExportInput struct {
	ConnectionID string
}

type BolExportResult struct {
	Pushed  int
	Skipped int // not enrolled in the connection's Bol sales channel
	Errors  []string
}

// RunScheduledBolExportUseCase is the runner cmd/scheduler calls for both
// the bol_products and bol_stock schedules (same use case, two schedule
// rows/intervals -- see domain/channels.SyncEntityBolProducts' doc comment
// for why: Bol has no stock-only endpoint, every push sends price+stock
// together). Enrollment-aware, unlike RunScheduledProductExportUseCase's
// Woo equivalent -- only pushes products matching this connection's Bol
// SalesChannel's enrollment rule (manual/category/tag + any per-product
// override), same matching EnrollProductInChannelsUseCase uses for the
// event-driven path, so a schedule never pushes something the merchant
// never opted into. Unlike Woo's scheduled export (which only catches
// products with no listing yet), this also re-pushes every already-linked
// enrolled product -- Bol has no separate stock-only call, so "refresh
// stock" and "catch stragglers" are the same PushProductToBolUseCase call.
type RunScheduledBolExportUseCase struct {
	db           *gorm.DB
	productRepo  pmsDomain.ProductRepository
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	pusher       bolProductPusher
}

func NewRunScheduledBolExportUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	pusher bolProductPusher,
) *RunScheduledBolExportUseCase {
	return &RunScheduledBolExportUseCase{
		db: db, productRepo: productRepo, channelRepo: channelRepo, overrideRepo: overrideRepo, pusher: pusher,
	}
}

func (uc *RunScheduledBolExportUseCase) Execute(ctx context.Context, in RunScheduledBolExportInput) (*BolExportResult, error) {
	var channel *channelsDomain.SalesChannel
	var products []pmsDomain.Product
	overrideByProduct := map[string]*channelsDomain.SalesChannelProductOverride{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		channels, err := uc.channelRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, ch := range channels {
			if ch.Type == channelsDomain.ChannelTypeBol && ch.Active && ch.ConnectionID != nil && *ch.ConnectionID == in.ConnectionID {
				c := ch
				channel = &c
				break
			}
		}
		if channel == nil {
			return nil
		}

		p, _, err := uc.productRepo.List(ctx, pmsDomain.ProductFilter{Limit: runScheduledBolExportLimit})
		if err != nil {
			return err
		}
		products = p

		// One query for every override on this channel instead of one
		// query per product -- the old per-product FindByChannelAndProduct
		// call meant every unenrolled product (the common case under
		// "manual only" enrollment, where most products have no override
		// row at all) logged a real-Postgres "record not found" line on
		// every scheduled run, for every product in the catalog.
		overrides, err := uc.overrideRepo.ListByChannel(ctx, channel.ID)
		if err != nil {
			return err
		}
		for i := range overrides {
			o := overrides[i]
			overrideByProduct[o.ProductID] = &o
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("runScheduledBolExport: %w", err)
	}

	result := &BolExportResult{}
	if channel == nil {
		// Not an error -- a Bol connection with no matching active
		// SalesChannel just has nothing to export yet. Surfaced in
		// result.Errors (not a silent 0-pushed success) so it shows up as
		// this schedule's LastRunError instead of looking like "ran fine,
		// nothing to do" when it's actually a missing setup step.
		result.Errors = append(result.Errors, fmt.Sprintf("no active Bol sales channel configured for connection %s -- nothing to export", in.ConnectionID))
		return result, nil
	}

	for _, p := range products {
		uc.exportOne(ctx, in, *channel, p, overrideByProduct[p.ID], result)
	}
	return result, nil
}

func (uc *RunScheduledBolExportUseCase) exportOne(ctx context.Context, in RunScheduledBolExportInput, channel channelsDomain.SalesChannel, p pmsDomain.Product, override *channelsDomain.SalesChannelProductOverride, result *BolExportResult) {
	if !channel.IsProductEnrolled(p.CategoryID, p.Tags, override) {
		result.Skipped++
		return
	}

	if _, err := uc.pusher.Execute(ctx, PushProductToBolInput{ProductID: p.ID, ConnectionID: in.ConnectionID}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", p.ID, err))
		return
	}
	result.Pushed++
}
