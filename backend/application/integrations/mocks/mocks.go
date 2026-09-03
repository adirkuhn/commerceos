package mocks

import (
	"context"
	"time"

	"github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/domain/shipping"
)

// ShippingConnectionSettingsRepo mock

type ShippingConnectionSettingsRepo struct {
	FindByConnectionIDFn func(ctx context.Context, connectionID string) (*shipping.ShippingConnectionSettings, error)
	UpsertFn             func(ctx context.Context, s *shipping.ShippingConnectionSettings) error
}

func (m *ShippingConnectionSettingsRepo) FindByConnectionID(ctx context.Context, connectionID string) (*shipping.ShippingConnectionSettings, error) {
	return m.FindByConnectionIDFn(ctx, connectionID)
}
func (m *ShippingConnectionSettingsRepo) Upsert(ctx context.Context, s *shipping.ShippingConnectionSettings) error {
	return m.UpsertFn(ctx, s)
}

// ConnectionRepo mock

type ConnectionRepo struct {
	CreateFn                  func(ctx context.Context, c *integrations.Connection) error
	FindByIDFn                func(ctx context.Context, id string) (*integrations.Connection, error)
	FindByStorefrontKeyHashFn func(ctx context.Context, hash string) (*integrations.Connection, error)
	ListByTenantFn            func(ctx context.Context) ([]integrations.Connection, error)
	UpdateFn                  func(ctx context.Context, c *integrations.Connection) error
	DeleteFn                  func(ctx context.Context, id string) error
}

func (m *ConnectionRepo) Create(ctx context.Context, c *integrations.Connection) error {
	return m.CreateFn(ctx, c)
}
func (m *ConnectionRepo) FindByID(ctx context.Context, id string) (*integrations.Connection, error) {
	return m.FindByIDFn(ctx, id)
}
func (m *ConnectionRepo) FindByStorefrontKeyHash(ctx context.Context, hash string) (*integrations.Connection, error) {
	return m.FindByStorefrontKeyHashFn(ctx, hash)
}
func (m *ConnectionRepo) ListByTenant(ctx context.Context) ([]integrations.Connection, error) {
	return m.ListByTenantFn(ctx)
}
func (m *ConnectionRepo) Update(ctx context.Context, c *integrations.Connection) error {
	return m.UpdateFn(ctx, c)
}
func (m *ConnectionRepo) Delete(ctx context.Context, id string) error {
	return m.DeleteFn(ctx, id)
}

// WooClientFactory mock

type WooClientFactory struct {
	NewClientFn func(storeURL, consumerKey, consumerSecret string) (integrations.WooClient, error)
}

func (m *WooClientFactory) NewClient(storeURL, consumerKey, consumerSecret string) (integrations.WooClient, error) {
	return m.NewClientFn(storeURL, consumerKey, consumerSecret)
}

// WooClient mock

type WooClient struct {
	TestConnectionFn func(ctx context.Context) error
}

func (m *WooClient) TestConnection(ctx context.Context) error {
	return m.TestConnectionFn(ctx)
}

// OutboxRepo mock

type OutboxRepo struct {
	EnqueueFn func(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

func (m *OutboxRepo) Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error {
	return m.EnqueueFn(ctx, subject, eventID, payload, occurredAt)
}

func (m *OutboxRepo) FetchPending(ctx context.Context, limit int) ([]outbox.Message, error) {
	panic("not implemented in mock")
}
func (m *OutboxRepo) MarkPublished(ctx context.Context, id uint) error {
	panic("not implemented in mock")
}
func (m *OutboxRepo) MarkFailed(ctx context.Context, id uint) error {
	panic("not implemented in mock")
}

// AIConnectionRepo mock

type AIConnectionRepo struct {
	GetFn    func(ctx context.Context) (*integrations.AIConnection, error)
	UpsertFn func(ctx context.Context, c *integrations.AIConnection) error
	DeleteFn func(ctx context.Context) error
}

func (m *AIConnectionRepo) Get(ctx context.Context) (*integrations.AIConnection, error) {
	return m.GetFn(ctx)
}
func (m *AIConnectionRepo) Upsert(ctx context.Context, c *integrations.AIConnection) error {
	return m.UpsertFn(ctx, c)
}
func (m *AIConnectionRepo) Delete(ctx context.Context) error {
	return m.DeleteFn(ctx)
}
