package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// createWooConnection writes the connection row + outbox event in one
// transaction — shared by the OAuth callback (CompleteWooConnectUseCase) and
// manual key-entry (ManualConnectWooCommerceUseCase) paths, which differ
// only in where tenantID/storeURL/credentials come from.
func createWooConnection(
	ctx context.Context,
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	outboxRepo outbox.Repository,
	tenantID, storeURL, consumerKey, encryptedSecret string,
) (*integrationsDomain.Connection, error) {
	ctx = tenant.WithTenantID(ctx, tenantID)

	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID:                tenantID,
			Provider:                integrationsDomain.ProviderWooCommerce,
			StoreURL:                integrationsDomain.StoreURL(storeURL),
			ConsumerKey:             consumerKey,
			ConsumerSecretEncrypted: encryptedSecret,
			Status:                  integrationsDomain.StatusConnected,
			CreatedAt:               now,
			UpdatedAt:               now,
		}
		if err := connRepo.Create(ctx, &conn); err != nil {
			return err
		}

		event := events.IntegrationConnected{
			Base: events.Base{
				EventID:    idgen.NewV4(),
				OccurredAt: now,
			},
			TenantID:     tenantID,
			Provider:     string(integrationsDomain.ProviderWooCommerce),
			ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, err
	}
	return &conn, nil
}
