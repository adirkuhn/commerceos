// Package whatsapp adapts the existing domain/integrations.EvolutionAPIClient
// (tenant-scoped WhatsApp send) to the generic domain/messaging.Client
// contract, without changing EvolutionAPIClient itself -- any future
// channel-agnostic "notify the customer" use case can depend on
// domain/messaging.Client instead of a WhatsApp-specific port.
package whatsapp

import (
	"context"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

type ClientAdapter struct {
	evo integrations.EvolutionAPIClient
}

func NewClientAdapter(evo integrations.EvolutionAPIClient) *ClientAdapter {
	return &ClientAdapter{evo: evo}
}

var _ messaging.Client = (*ClientAdapter)(nil)

func (a *ClientAdapter) Send(ctx context.Context, msg messaging.Message) error {
	if msg.Channel != messaging.ChannelWhatsApp {
		return fmt.Errorf("whatsapp.ClientAdapter.Send: unsupported channel %q", msg.Channel)
	}
	return a.evo.SendText(ctx, msg.To, msg.Body)
}
