package channels

import (
	"context"
	"log"

	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// DeleteVariantFromChannelsUseCase takes one unlisted variant off every
// external channel it was on (see events.VariantUnlisted): its standalone
// Woo product, or its variation under the parent variable product. Best
// effort per listing, same contract as DeleteProductFromChannelsUseCase --
// a failure is logged and the rest still go.
type DeleteVariantFromChannelsUseCase struct {
	clients wooClientResolver
}

func NewDeleteVariantFromChannelsUseCase(clients wooClientResolver) *DeleteVariantFromChannelsUseCase {
	return &DeleteVariantFromChannelsUseCase{clients: clients}
}

func (uc *DeleteVariantFromChannelsUseCase) Execute(ctx context.Context, listings []events.VariantUnlistedListing) error {
	for _, l := range listings {
		if l.Provider != string(integrationsDomain.ProviderWooCommerce) {
			continue // no adapter for any other provider yet
		}
		if err := uc.deleteOne(ctx, l); err != nil {
			log.Printf("deleteVariantFromChannels: connection %s external %s: %v", l.ConnectionID, l.ExternalID, err)
		}
	}
	return nil
}

func (uc *DeleteVariantFromChannelsUseCase) deleteOne(ctx context.Context, l events.VariantUnlistedListing) error {
	client, err := uc.clients.ClientFor(ctx, l.ConnectionID)
	if err != nil {
		return err
	}
	if l.ParentExternalID != "" {
		return client.DeleteVariation(ctx, l.ParentExternalID, l.ExternalID)
	}
	return client.DeleteProduct(ctx, l.ExternalID)
}
