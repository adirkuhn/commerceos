package shipping

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// shippingProviderSupport reports whether a provider's connections can
// create shipping labels -- infrastructure/carriers.Registry satisfies it.
type shippingProviderSupport interface {
	Supports(provider integrationsDomain.Provider) bool
}

// MissingSenderAddress: the connection's carrier needs a sender address
// and the tenant has none (no shipping sender, no company address).
const MissingSenderAddress = "sender_address"

// ShippingConnection is a shipping connection plus what's still missing
// before it can create labels (empty = ready).
type ShippingConnection struct {
	integrationsDomain.Connection
	Missing []string
}

// ShippingConnections is every connection packing can create a label
// through, oldest first, plus which one it uses by default.
type ShippingConnections struct {
	Connections []ShippingConnection
	// DefaultConnectionID is the one packing preselects -- "" only when
	// there are no shipping connections at all.
	DefaultConnectionID string
	// ConfiguredConnectionID is the merchant's explicit pick, nil when
	// none was made (or it was cleared).
	ConfiguredConnectionID *string
}

type ListShippingConnectionsUseCase struct {
	db        *gorm.DB
	connRepo  integrationsDomain.ConnectionRepository
	prefsRepo shippingDomain.ShippingPreferencesRepository
	support   shippingProviderSupport
	addresses shipmentAddressSource
}

func NewListShippingConnectionsUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, prefsRepo shippingDomain.ShippingPreferencesRepository, support shippingProviderSupport, addresses shipmentAddressSource) *ListShippingConnectionsUseCase {
	return &ListShippingConnectionsUseCase{db: db, connRepo: connRepo, prefsRepo: prefsRepo, support: support, addresses: addresses}
}

func (uc *ListShippingConnectionsUseCase) Execute(ctx context.Context) (*ShippingConnections, error) {
	var result ShippingConnections
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		all, err := uc.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		prefs, err := uc.prefsRepo.Get(ctx)
		if err != nil {
			return err
		}
		if prefs != nil {
			result.ConfiguredConnectionID = prefs.DefaultConnectionID
		}
		addresses, err := uc.addresses.Resolve(ctx)
		if err != nil {
			return err
		}
		for _, c := range shippingConnectionsOf(all, uc.support) {
			result.Connections = append(result.Connections, ShippingConnection{Connection: c, Missing: missingFor(c, addresses)})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listShippingConnections: %w", err)
	}
	candidates := make([]shippingDomain.ConnectionCandidate, len(result.Connections))
	for i, c := range result.Connections {
		candidates[i] = shippingDomain.ConnectionCandidate{ID: c.ID, CreatedAt: c.CreatedAt}
	}
	result.DefaultConnectionID = shippingDomain.ResolveDefaultConnection(candidates, result.ConfiguredConnectionID)
	return &result, nil
}

func missingFor(c integrationsDomain.Connection, addresses ShipmentAddresses) []string {
	if c.Provider.RequiresSenderAddress() && addresses.Sender == nil {
		return []string{MissingSenderAddress}
	}
	return nil
}

func shippingConnectionsOf(all []integrationsDomain.Connection, support shippingProviderSupport) []integrationsDomain.Connection {
	out := make([]integrationsDomain.Connection, 0, len(all))
	for _, c := range all {
		if support.Supports(c.Provider) {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}

type SetDefaultShippingConnectionUseCase struct {
	db        *gorm.DB
	connRepo  integrationsDomain.ConnectionRepository
	prefsRepo shippingDomain.ShippingPreferencesRepository
	support   shippingProviderSupport
	outbox    eventrecord.Enqueuer
}

func NewSetDefaultShippingConnectionUseCase(db *gorm.DB, connRepo integrationsDomain.ConnectionRepository, prefsRepo shippingDomain.ShippingPreferencesRepository, support shippingProviderSupport, outbox eventrecord.Enqueuer) *SetDefaultShippingConnectionUseCase {
	return &SetDefaultShippingConnectionUseCase{db: db, connRepo: connRepo, prefsRepo: prefsRepo, support: support, outbox: outbox}
}

// Execute makes connectionID packing's default shipping connection; nil
// clears the choice (the oldest shipping connection is used again).
func (uc *SetDefaultShippingConnectionUseCase) Execute(ctx context.Context, connectionID *string) error {
	if !tenant.IsTenantAdmin(ctx) {
		return fmt.Errorf("setDefaultShippingConnection: %w", shippingDomain.ErrShippingPreferencesForbidden)
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if connectionID != nil {
			conn, err := uc.connRepo.FindByID(ctx, *connectionID)
			if err != nil {
				return err
			}
			if !uc.support.Supports(conn.Provider) {
				return shippingDomain.ErrNotAShippingConnection
			}
		}
		before, err := uc.prefsRepo.Get(ctx)
		if err != nil {
			return err
		}
		next := shippingDomain.ShippingPreferences{}
		if before != nil {
			next = *before
		}
		beforeID := next.DefaultConnectionID
		tenantID := tenant.IDFromContext(ctx)
		next.TenantID, next.DefaultConnectionID, next.UpdatedAt = tenantID, connectionID, time.Now()
		if err := uc.prefsRepo.Upsert(ctx, &next); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ShippingPreferencesUpdated, tenantID,
			eventrecord.NewChanges().Set("default_connection_id", beforeID, connectionID))
	})
	if err != nil {
		return fmt.Errorf("setDefaultShippingConnection: %w", err)
	}
	return nil
}
