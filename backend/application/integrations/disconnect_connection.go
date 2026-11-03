package integrations

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DisconnectConnectionUseCase struct {
	db       *gorm.DB
	connRepo integrationsDomain.ConnectionRepository
	outbox   eventrecord.Enqueuer
}

func NewDisconnectConnectionUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, outbox eventrecord.Enqueuer) *DisconnectConnectionUseCase {
	return &DisconnectConnectionUseCase{db: db, connRepo: connRepo, outbox: outbox}
}

func (uc *DisconnectConnectionUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conn, err := uc.connRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		// Never delete the row: its FKs cascade into sales channels,
		// listings, schedules and category links (COM-15).
		conn.Disconnect(time.Now())
		if err := uc.connRepo.Update(ctx, conn); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ConnectionDisconnected, id, eventrecord.Data{"provider": string(conn.Provider)})
	})
	if err != nil {
		return fmt.Errorf("disconnectConnection: %w", err)
	}
	return nil
}
