// Package webhooks holds the use cases for inbound third-party webhook
// ingestion: verify + publish on the hot path, and the admin triage
// actions (list/retry/abandon) over whatever a publish failure left behind.
package webhooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Publisher is JetStream narrowed to point of use (ISP) -- lets
// ReceiveWebhookUseCase be unit tested without a real NATS connection. The
// real adapter (infrastructure/publishers.JetStreamPublisher) publishes
// with an ack, so a nil error here means JetStream durably has the
// message.
type Publisher interface {
	Publish(subject string, data []byte) error
}

// maxPublishRetries bounds ReceiveWebhookUseCase's in-request retry of a
// failed JetStream publish -- must stay small: this runs inside the HTTP
// handler, so unbounded retries would just turn a NATS outage into request
// timeouts instead of the fast, deliberate fallback-table write in step 4
// below. 3 is the number the user asked for.
const maxPublishRetries = 3

const webhookSubjectPrefix = "webhooks.received."

// Result reports what ReceiveWebhookUseCase actually did with the
// request, so the handler can pick the right HTTP status: 200 when
// JetStream durably has it, 202 when it only made it as far as the
// fallback table (still durably saved, just not yet queued for
// processing).
type Result struct {
	EventID      string
	FallbackUsed bool
}

// ReceiveWebhookUseCase is the hot path: verify the provider's signature,
// then publish to COMMERCEOS_WEBHOOKS. A provider unknown to the registry
// or lacking a configured secret never reaches JetStream or the DB at all.
type ReceiveWebhookUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	verifiers        map[string]webhooks.ProviderVerifier
	publisher        Publisher
	fallbackRepo     webhooks.FallbackEventRepository
	encryptionSecret string
}

func NewReceiveWebhookUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	verifiers map[string]webhooks.ProviderVerifier,
	publisher Publisher,
	fallbackRepo webhooks.FallbackEventRepository,
	encryptionSecret string,
) *ReceiveWebhookUseCase {
	return &ReceiveWebhookUseCase{
		db: db, connRepo: connRepo, verifiers: verifiers, publisher: publisher,
		fallbackRepo: fallbackRepo, encryptionSecret: encryptionSecret,
	}
}

func (uc *ReceiveWebhookUseCase) Execute(ctx context.Context, tenantID, provider string, rawBody []byte, headers map[string]string) (Result, error) {
	verifier, ok := uc.verifiers[provider]
	if !ok {
		return Result{}, webhooks.ErrProviderUnknown
	}

	ctx = tenant.WithTenantID(ctx, tenantID)

	secret, err := uc.tenantWebhookSecret(ctx, provider)
	if err != nil {
		return Result{}, fmt.Errorf("receiveWebhook: load secret: %w", err)
	}
	if secret == "" {
		return Result{}, webhooks.ErrProviderNotImplemented
	}

	if err := verifier.Verify(rawBody, headers, secret); err != nil {
		return Result{}, err
	}

	eventID := verifier.EventID(headers, rawBody)
	if eventID == "" {
		sum := sha256.Sum256(rawBody)
		eventID = hex.EncodeToString(sum[:])
	}

	envelope := webhooks.Envelope{
		EventID: eventID, TenantID: tenantID, Provider: provider,
		ReceivedAt: time.Now().UTC(), RawPayload: rawBody, Headers: headers,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return Result{}, fmt.Errorf("receiveWebhook: marshal envelope: %w", err)
	}
	subject := webhookSubjectPrefix + provider

	var publishErr error
	for attempt := 0; attempt < maxPublishRetries; attempt++ {
		if publishErr = uc.publisher.Publish(subject, data); publishErr == nil {
			return Result{EventID: eventID}, nil
		}
		// A too-large payload fails identically on every attempt -- retrying
		// the same bytes 2 more times just delays the fallback save for no
		// chance of a different outcome.
		if errors.Is(publishErr, webhooks.ErrPayloadTooLarge) {
			break
		}
	}

	// Every publish attempt failed -- JetStream unreachable or refusing the
	// message. Durably save it ourselves rather than lose it.
	fallbackErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.fallbackRepo.Save(ctx, &webhooks.FallbackEvent{
			TenantID: tenantID, Provider: provider, EventID: eventID,
			Payload: rawBody, Headers: headers,
			Status: webhooks.FallbackStatusPendingRepublish, ReceivedAt: envelope.ReceivedAt,
		})
	})
	if fallbackErr != nil {
		return Result{}, fmt.Errorf("receiveWebhook: publish failed (%w) and fallback save failed: %w", publishErr, fallbackErr)
	}
	return Result{EventID: eventID, FallbackUsed: true}, nil
}

// tenantWebhookSecret looks up the tenant's connection for provider and
// decrypts its webhook secret. Returns "" (no error) when no connection
// exists for that provider, or a connection exists but has no secret
// configured yet -- both are "not implemented for this tenant yet", not a
// fault.
func (uc *ReceiveWebhookUseCase) tenantWebhookSecret(ctx context.Context, provider string) (string, error) {
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		conns, err := uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, c := range conns {
			if string(c.Provider) != provider || c.WebhookSecretEncrypted == "" {
				continue
			}
			decrypted, err := crypto.Decrypt(uc.encryptionSecret, c.WebhookSecretEncrypted)
			if err != nil {
				return fmt.Errorf("decrypt webhook secret: %w", err)
			}
			secret = decrypted
			return nil
		}
		return nil
	})
	return secret, err
}
