package integrations

import (
	"context"
	"fmt"
	"net/url"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// providerWebhookIngestURL builds the base callback URL a provider should
// deliver to -- CommerceOS's own unauthenticated webhook-ingestion route
// (see api/webhooks/openapi.yaml). WooCommerce signs deliveries via the
// request body's own `secret` field (see CreateProviderWebhookUseCase).
func providerWebhookIngestURL(publicAPIBaseURL, tenantID string, provider integrationsDomain.Provider) string {
	return publicAPIBaseURL + "/api/v1/webhooks/v1/" + url.PathEscape(tenantID) + "/" + string(provider)
}

// providerWebhookIngestURLWithToken is providerWebhookIngestURL for a
// provider with no signature scheme of its own (MyParcel) -- the shared
// secret travels as a path segment (api/webhooks/openapi.yaml's
// receiveWebhookWithToken route) rather than a `?token=` query string,
// which is more likely to leak into access logs/proxies/browser history.
// Same convention application/shipping.RegisterMyParcelWebhookUseCase uses.
func providerWebhookIngestURLWithToken(publicAPIBaseURL, tenantID string, provider integrationsDomain.Provider, token string) string {
	return providerWebhookIngestURL(publicAPIBaseURL, tenantID, provider) + "/" + url.PathEscape(token)
}

// ListProviderWebhooksUseCase lists what's actually registered at the
// provider for this connection -- distinct from
// application/webhooks.ListForwardRulesUseCase, which lists CommerceOS's
// own fan-out configuration on top of an event once received.
type ListProviderWebhooksUseCase struct {
	db              *gorm.DB
	connRepo        integrationsDomain.ConnectionRepository
	wooFactory      integrationsDomain.WooClientFactory
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
}

func NewListProviderWebhooksUseCase(
	db *gorm.DB, connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory, myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
) *ListProviderWebhooksUseCase {
	return &ListProviderWebhooksUseCase{db: db, connRepo: connRepo, wooFactory: wooFactory, myParcelFactory: myParcelFactory, encSecret: encryptionSecret}
}

func (uc *ListProviderWebhooksUseCase) Execute(ctx context.Context, connectionID string) ([]integrationsDomain.ProviderWebhook, error) {
	var out []integrationsDomain.ProviderWebhook
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conn, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		switch conn.Provider {
		case integrationsDomain.ProviderWooCommerce:
			client, err := newWooClientForConn(uc.wooFactory, uc.encSecret, *conn)
			if err != nil {
				return err
			}
			hooks, err := client.ListWebhooks(ctx)
			if err != nil {
				return fmt.Errorf("%w: %w", integrationsDomain.ErrProviderWebhookRequestFailed, err)
			}
			out = make([]integrationsDomain.ProviderWebhook, len(hooks))
			for i, h := range hooks {
				out[i] = integrationsDomain.ProviderWebhook{ID: h.ID, Provider: conn.Provider, Topic: h.Topic, DeliveryURL: h.DeliveryURL, Status: h.Status}
			}
			return nil
		case integrationsDomain.ProviderMyParcel:
			client, err := newMyParcelClientForConn(uc.myParcelFactory, uc.encSecret, *conn)
			if err != nil {
				return err
			}
			subs, err := client.ListWebhookSubscriptions(ctx)
			if err != nil {
				return fmt.Errorf("%w: %w", integrationsDomain.ErrProviderWebhookRequestFailed, err)
			}
			out = make([]integrationsDomain.ProviderWebhook, len(subs))
			for i, s := range subs {
				out[i] = integrationsDomain.ProviderWebhook{ID: s.ID, Provider: conn.Provider, Topic: s.Hook, DeliveryURL: s.URL, Status: "active"}
			}
			return nil
		default:
			return integrationsDomain.ErrProviderNotSupported
		}
	})
	if err != nil {
		return nil, fmt.Errorf("listProviderWebhooks: %w", err)
	}
	return out, nil
}

// ListAllProviderWebhooksResult mirrors ReconcileShipmentsResult's
// partial-failure-tolerant shape: one connection's live call failing (a
// revoked key, a store that's down) must not hide every other
// connection's subscriptions -- callers see both the rows that succeeded
// and which connections failed, with why.
type ListAllProviderWebhooksResult struct {
	Webhooks []integrationsDomain.ProviderWebhook
	Errors   []string
}

// ListAllProviderWebhooksUseCase aggregates ListProviderWebhooksUseCase
// across every WooCommerce/MyParcel connection the tenant has -- the "one
// place to see everything actually subscribed" view, instead of having to
// open each connection's own settings page one at a time.
type ListAllProviderWebhooksUseCase struct {
	db              *gorm.DB
	connRepo        integrationsDomain.ConnectionRepository
	wooFactory      integrationsDomain.WooClientFactory
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
}

func NewListAllProviderWebhooksUseCase(
	db *gorm.DB, connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory, myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
) *ListAllProviderWebhooksUseCase {
	return &ListAllProviderWebhooksUseCase{db: db, connRepo: connRepo, wooFactory: wooFactory, myParcelFactory: myParcelFactory, encSecret: encryptionSecret}
}

func (uc *ListAllProviderWebhooksUseCase) Execute(ctx context.Context) (*ListAllProviderWebhooksResult, error) {
	var conns []integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		conns, err = uc.connRepo.ListByTenant(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listAllProviderWebhooks: %w", err)
	}

	result := &ListAllProviderWebhooksResult{}
	for _, conn := range conns {
		switch conn.Provider {
		case integrationsDomain.ProviderWooCommerce:
			client, err := newWooClientForConn(uc.wooFactory, uc.encSecret, conn)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %v", conn.Provider, conn.ID, err))
				continue
			}
			hooks, err := client.ListWebhooks(ctx)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %v", conn.Provider, conn.ID, err))
				continue
			}
			for _, h := range hooks {
				result.Webhooks = append(result.Webhooks, integrationsDomain.ProviderWebhook{
					ID: h.ID, Provider: conn.Provider, Topic: h.Topic, DeliveryURL: h.DeliveryURL, Status: h.Status,
					ConnectionID: conn.ID, StoreLabel: conn.StoreURL.String(),
				})
			}
		case integrationsDomain.ProviderMyParcel:
			client, err := newMyParcelClientForConn(uc.myParcelFactory, uc.encSecret, conn)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %v", conn.Provider, conn.ID, err))
				continue
			}
			subs, err := client.ListWebhookSubscriptions(ctx)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %v", conn.Provider, conn.ID, err))
				continue
			}
			for _, s := range subs {
				result.Webhooks = append(result.Webhooks, integrationsDomain.ProviderWebhook{
					ID: s.ID, Provider: conn.Provider, Topic: s.Hook, DeliveryURL: s.URL, Status: "active",
					ConnectionID: conn.ID, StoreLabel: "MyParcel",
				})
			}
		default:
			// Bol and any other connected-but-unsupported provider: not an
			// error, just nothing to list -- ListProviderWebhooksUseCase
			// returns ErrProviderNotSupported for a direct per-connection
			// call, but silently skipping here keeps this aggregate view
			// from showing an error row for every non-webhook-capable
			// connection a tenant happens to have.
		}
	}
	return result, nil
}

// CreateProviderWebhookUseCase registers a new subscription for topic,
// pointed at CommerceOS's own ingest URL -- the "subscribe CommerceOS to
// receive this" action. Requires the connection's WebhookSecretEncrypted
// to already be set (SetWebhookSecretUseCase) -- both providers need it to
// let CommerceOS verify what it's about to start receiving, just via
// different mechanisms (see providerWebhookIngestURL's doc comment).
type CreateProviderWebhookUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	myParcelFactory  integrationsDomain.MyParcelClientFactory
	encSecret        string
	publicAPIBaseURL string
	outbox           eventrecord.Enqueuer
}

func NewCreateProviderWebhookUseCase(
	db *gorm.DB, connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory, myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret, publicAPIBaseURL string,
	outbox eventrecord.Enqueuer,
) *CreateProviderWebhookUseCase {
	return &CreateProviderWebhookUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, myParcelFactory: myParcelFactory,
		encSecret: encryptionSecret, publicAPIBaseURL: publicAPIBaseURL, outbox: outbox,
	}
}

func (uc *CreateProviderWebhookUseCase) Execute(ctx context.Context, connectionID, topic string) (*integrationsDomain.ProviderWebhook, error) {
	var out integrationsDomain.ProviderWebhook
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conn, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		if conn.WebhookSecretEncrypted == "" {
			return integrationsDomain.ErrWebhookSecretNotSet
		}
		webhookSecret, err := crypto.Decrypt(uc.encSecret, conn.WebhookSecretEncrypted)
		if err != nil {
			return fmt.Errorf("decrypt webhook secret: %w", err)
		}

		switch conn.Provider {
		case integrationsDomain.ProviderWooCommerce:
			client, err := newWooClientForConn(uc.wooFactory, uc.encSecret, *conn)
			if err != nil {
				return err
			}
			deliveryURL := providerWebhookIngestURL(uc.publicAPIBaseURL, conn.TenantID, conn.Provider)
			hook, err := client.CreateWebhook(ctx, topic, deliveryURL, webhookSecret)
			if err != nil {
				return fmt.Errorf("%w: %w", integrationsDomain.ErrProviderWebhookRequestFailed, err)
			}
			out = integrationsDomain.ProviderWebhook{ID: hook.ID, Provider: conn.Provider, Topic: hook.Topic, DeliveryURL: hook.DeliveryURL, Status: hook.Status}
			return recordProviderWebhookCreated(ctx, uc.outbox, conn.ID, out)
		case integrationsDomain.ProviderMyParcel:
			client, err := newMyParcelClientForConn(uc.myParcelFactory, uc.encSecret, *conn)
			if err != nil {
				return err
			}
			deliveryURL := providerWebhookIngestURLWithToken(uc.publicAPIBaseURL, conn.TenantID, conn.Provider, webhookSecret)
			sub, err := client.CreateWebhookSubscription(ctx, topic, deliveryURL)
			if err != nil {
				return fmt.Errorf("%w: %w", integrationsDomain.ErrProviderWebhookRequestFailed, err)
			}
			out = integrationsDomain.ProviderWebhook{ID: sub.ID, Provider: conn.Provider, Topic: sub.Hook, DeliveryURL: sub.URL, Status: "active"}
			return recordProviderWebhookCreated(ctx, uc.outbox, conn.ID, out)
		default:
			return integrationsDomain.ErrProviderNotSupported
		}
	})
	if err != nil {
		return nil, fmt.Errorf("createProviderWebhook: %w", err)
	}
	return &out, nil
}

// DeleteProviderWebhookUseCase removes one subscription by the provider's
// own id -- unregisters CommerceOS from receiving it.
type DeleteProviderWebhookUseCase struct {
	db              *gorm.DB
	connRepo        integrationsDomain.ConnectionRepository
	wooFactory      integrationsDomain.WooClientFactory
	myParcelFactory integrationsDomain.MyParcelClientFactory
	encSecret       string
	outbox          eventrecord.Enqueuer
}

func NewDeleteProviderWebhookUseCase(
	db *gorm.DB, connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory, myParcelFactory integrationsDomain.MyParcelClientFactory,
	encryptionSecret string,
	outbox eventrecord.Enqueuer,
) *DeleteProviderWebhookUseCase {
	return &DeleteProviderWebhookUseCase{db: db, connRepo: connRepo, wooFactory: wooFactory, myParcelFactory: myParcelFactory, encSecret: encryptionSecret, outbox: outbox}
}

func (uc *DeleteProviderWebhookUseCase) Execute(ctx context.Context, connectionID, webhookID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conn, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		switch conn.Provider {
		case integrationsDomain.ProviderWooCommerce:
			client, err := newWooClientForConn(uc.wooFactory, uc.encSecret, *conn)
			if err != nil {
				return err
			}
			if err := client.DeleteWebhook(ctx, webhookID); err != nil {
				return fmt.Errorf("%w: %w", integrationsDomain.ErrProviderWebhookRequestFailed, err)
			}
			return recordProviderWebhookDeleted(ctx, uc.outbox, *conn, webhookID)
		case integrationsDomain.ProviderMyParcel:
			client, err := newMyParcelClientForConn(uc.myParcelFactory, uc.encSecret, *conn)
			if err != nil {
				return err
			}
			if err := client.DeleteWebhookSubscription(ctx, webhookID); err != nil {
				return fmt.Errorf("%w: %w", integrationsDomain.ErrProviderWebhookRequestFailed, err)
			}
			return recordProviderWebhookDeleted(ctx, uc.outbox, *conn, webhookID)
		default:
			return integrationsDomain.ErrProviderNotSupported
		}
	})
	if err != nil {
		return fmt.Errorf("deleteProviderWebhook: %w", err)
	}
	return nil
}

func newWooClientForConn(factory integrationsDomain.WooClientFactory, encSecret string, conn integrationsDomain.Connection) (integrationsDomain.WooClient, error) {
	secret, err := crypto.Decrypt(encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt consumer secret: %w", err)
	}
	client, err := factory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("build woo client: %w", err)
	}
	return client, nil
}

func newMyParcelClientForConn(factory integrationsDomain.MyParcelClientFactory, encSecret string, conn integrationsDomain.Connection) (integrationsDomain.MyParcelClient, error) {
	apiKey, err := crypto.Decrypt(encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key: %w", err)
	}
	client, err := factory.NewClient(apiKey)
	if err != nil {
		return nil, fmt.Errorf("build myparcel client: %w", err)
	}
	return client, nil
}

func recordProviderWebhookCreated(ctx context.Context, outbox eventrecord.Enqueuer, connectionID string, hook integrationsDomain.ProviderWebhook) error {
	return eventrecord.Record(ctx, outbox, events.ProviderWebhookCreated, hook.ID, eventrecord.Data{
		"connection_id": connectionID, "provider": string(hook.Provider), "topic": hook.Topic,
	})
}

func recordProviderWebhookDeleted(ctx context.Context, outbox eventrecord.Enqueuer, conn integrationsDomain.Connection, webhookID string) error {
	return eventrecord.Record(ctx, outbox, events.ProviderWebhookDeleted, webhookID, eventrecord.Data{
		"connection_id": conn.ID, "provider": string(conn.Provider),
	})
}
