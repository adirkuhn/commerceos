package integrations

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// secretChanged stands in for a credential's value in an event's changes:
// that it changed is recorded, never the value.
const secretChanged = "changed"

// recordConnectionUpdated records integrations.connection.updated with
// the given changes.
func recordConnectionUpdated(ctx context.Context, outbox eventrecord.Enqueuer, conn integrationsDomain.Connection, changes eventrecord.Changes) error {
	return eventrecord.Record(ctx, outbox, events.ConnectionUpdated, conn.ID,
		eventrecord.Data{"provider": string(conn.Provider)}, changes)
}
