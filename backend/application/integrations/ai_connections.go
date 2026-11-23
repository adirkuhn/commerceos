package integrations

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// The tenant's AI connections are an ordered list: every AI feature routes
// through it, trying the first enabled connection and moving to the next on
// any failure (see infrastructure/ai's router).

// ListAIConnectionsUseCase returns the connections in routing order.
type ListAIConnectionsUseCase struct {
	db   *gorm.DB
	repo integrationsDomain.AIConnectionRepository
}

func NewListAIConnectionsUseCase(db *gorm.DB, repo integrationsDomain.AIConnectionRepository) *ListAIConnectionsUseCase {
	return &ListAIConnectionsUseCase{db: db, repo: repo}
}

func (uc *ListAIConnectionsUseCase) Execute(ctx context.Context) ([]integrationsDomain.AIConnection, error) {
	var out []integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.repo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listAIConnections: %w", err)
	}
	return out, nil
}

// AIConnectionInput is a new connection. APIKey is plaintext in, encrypted
// before it's stored; empty only for Ollama.
type AIConnectionInput struct {
	Provider       integrationsDomain.AIProvider
	APIKey         string
	BaseURL        string
	Model          string
	FallbackModels string
	Label          string
	// Enabled nil means enabled.
	Enabled *bool
}

// CreateAIConnectionUseCase adds a connection at the end of the list.
type CreateAIConnectionUseCase struct {
	db               *gorm.DB
	repo             integrationsDomain.AIConnectionRepository
	encryptionSecret string
	outbox           eventrecord.Enqueuer
}

func NewCreateAIConnectionUseCase(db *gorm.DB, repo integrationsDomain.AIConnectionRepository, encryptionSecret string, outbox eventrecord.Enqueuer) *CreateAIConnectionUseCase {
	return &CreateAIConnectionUseCase{db: db, repo: repo, encryptionSecret: encryptionSecret, outbox: outbox}
}

func (uc *CreateAIConnectionUseCase) Execute(ctx context.Context, in AIConnectionInput) (*integrationsDomain.AIConnection, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("createAIConnection: %w", integrationsDomain.ErrNoTenantContext)
	}
	keyEncrypted, err := encryptKey(uc.encryptionSecret, in.APIKey)
	if err != nil {
		return nil, fmt.Errorf("createAIConnection: %w", err)
	}
	now := time.Now()
	c := &integrationsDomain.AIConnection{
		TenantID: tenantID, Provider: in.Provider, APIKeyEncrypted: keyEncrypted, BaseURL: in.BaseURL, Model: in.Model,
		FallbackModels: in.FallbackModels, Label: in.Label, Enabled: in.Enabled == nil || *in.Enabled,
		Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("createAIConnection: %w", err)
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.List(ctx)
		if err != nil {
			return err
		}
		c.Position = integrationsDomain.NextAIPosition(existing)
		if err := uc.repo.Create(ctx, c); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AIConnectionConnected, c.ID, eventrecord.Data{
			"provider": string(c.Provider), "model": c.Model, "base_url": c.BaseURL, "fallback_models": c.FallbackModels,
			"label": c.Label, "position": c.Position, "enabled": c.Enabled,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("createAIConnection: %w", err)
	}
	return c, nil
}

// AIConnectionPatch changes only the fields that are set. APIKey nil or ""
// keeps the stored key (it never round-trips to the client).
type AIConnectionPatch struct {
	Provider       *integrationsDomain.AIProvider
	APIKey         *string
	BaseURL        *string
	Model          *string
	FallbackModels *string
	Label          *string
	Enabled        *bool
}

// UpdateAIConnectionUseCase edits one connection in place (its position is
// changed only by ReorderAIConnectionsUseCase).
type UpdateAIConnectionUseCase struct {
	db               *gorm.DB
	repo             integrationsDomain.AIConnectionRepository
	encryptionSecret string
	outbox           eventrecord.Enqueuer
}

func NewUpdateAIConnectionUseCase(db *gorm.DB, repo integrationsDomain.AIConnectionRepository, encryptionSecret string, outbox eventrecord.Enqueuer) *UpdateAIConnectionUseCase {
	return &UpdateAIConnectionUseCase{db: db, repo: repo, encryptionSecret: encryptionSecret, outbox: outbox}
}

func (uc *UpdateAIConnectionUseCase) Execute(ctx context.Context, id string, p AIConnectionPatch) (*integrationsDomain.AIConnection, error) {
	var out *integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		before := *c
		keyChanged, err := uc.apply(c, p)
		if err != nil {
			return err
		}
		if err := c.Validate(); err != nil {
			return err
		}
		c.UpdatedAt = time.Now()
		if err := uc.repo.Update(ctx, c); err != nil {
			return err
		}
		out = c
		opts := []eventrecord.Option{aiConnectionChanges(before, *c)}
		if keyChanged {
			opts = append(opts, eventrecord.Data{"api_key_changed": true})
		}
		return eventrecord.Record(ctx, uc.outbox, events.AIConnectionUpdated, c.ID, opts...)
	})
	if err != nil {
		return nil, fmt.Errorf("updateAIConnection: %w", err)
	}
	return out, nil
}

func (uc *UpdateAIConnectionUseCase) apply(c *integrationsDomain.AIConnection, p AIConnectionPatch) (keyChanged bool, err error) {
	set := func(dst *string, src *string) {
		if src != nil {
			*dst = *src
		}
	}
	if p.Provider != nil {
		c.Provider = *p.Provider
	}
	set(&c.BaseURL, p.BaseURL)
	set(&c.Model, p.Model)
	set(&c.FallbackModels, p.FallbackModels)
	set(&c.Label, p.Label)
	if p.Enabled != nil {
		c.Enabled = *p.Enabled
	}
	if p.APIKey != nil && *p.APIKey != "" {
		if c.APIKeyEncrypted, err = encryptKey(uc.encryptionSecret, *p.APIKey); err != nil {
			return false, err
		}
		keyChanged = true
	}
	return keyChanged, nil
}

func aiConnectionChanges(before, after integrationsDomain.AIConnection) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("provider", string(before.Provider), string(after.Provider)).
		Set("model", before.Model, after.Model).
		Set("base_url", before.BaseURL, after.BaseURL).
		Set("fallback_models", before.FallbackModels, after.FallbackModels).
		Set("label", before.Label, after.Label).
		Set("enabled", before.Enabled, after.Enabled)
}

// DeleteAIConnectionUseCase removes a connection and closes the gap it
// leaves in the order.
type DeleteAIConnectionUseCase struct {
	db     *gorm.DB
	repo   integrationsDomain.AIConnectionRepository
	outbox eventrecord.Enqueuer
}

func NewDeleteAIConnectionUseCase(db *gorm.DB, repo integrationsDomain.AIConnectionRepository, outbox eventrecord.Enqueuer) *DeleteAIConnectionUseCase {
	return &DeleteAIConnectionUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *DeleteAIConnectionUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.repo.Delete(ctx, id); err != nil {
			return err
		}
		rest, err := uc.repo.List(ctx)
		if err != nil {
			return err
		}
		if err := savePositions(ctx, uc.repo, rest, integrationsDomain.CompactAIPositions(rest)); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.AIConnectionDisconnected, c.ID, eventrecord.Data{"provider": string(c.Provider)})
	})
	if err != nil {
		return fmt.Errorf("deleteAIConnection: %w", err)
	}
	return nil
}

// ReorderAIConnectionsUseCase sets the routing order: ids lists every
// connection once, first tried first.
type ReorderAIConnectionsUseCase struct {
	db     *gorm.DB
	repo   integrationsDomain.AIConnectionRepository
	outbox eventrecord.Enqueuer
}

func NewReorderAIConnectionsUseCase(db *gorm.DB, repo integrationsDomain.AIConnectionRepository, outbox eventrecord.Enqueuer) *ReorderAIConnectionsUseCase {
	return &ReorderAIConnectionsUseCase{db: db, repo: repo, outbox: outbox}
}

func (uc *ReorderAIConnectionsUseCase) Execute(ctx context.Context, ids []string) ([]integrationsDomain.AIConnection, error) {
	var out []integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.repo.List(ctx)
		if err != nil {
			return err
		}
		reordered, err := integrationsDomain.ReorderAIConnections(existing, ids)
		if err != nil {
			return err
		}
		if err := savePositions(ctx, uc.repo, existing, reordered); err != nil {
			return err
		}
		out = reordered
		return eventrecord.Record(ctx, uc.outbox, events.AIConnectionsReordered, tenant.IDFromContext(ctx), eventrecord.Data{"order": ids})
	})
	if err != nil {
		return nil, fmt.Errorf("reorderAIConnections: %w", err)
	}
	return out, nil
}

// savePositions writes the connections whose position changed.
func savePositions(ctx context.Context, repo integrationsDomain.AIConnectionRepository, before, after []integrationsDomain.AIConnection) error {
	was := make(map[string]int, len(before))
	for _, c := range before {
		was[c.ID] = c.Position
	}
	for i := range after {
		if pos, ok := was[after[i].ID]; ok && pos == after[i].Position {
			continue
		}
		if err := repo.Update(ctx, &after[i]); err != nil {
			return err
		}
	}
	return nil
}

func encryptKey(secret, apiKey string) (string, error) {
	if apiKey == "" {
		return "", nil
	}
	return crypto.Encrypt(secret, apiKey)
}
