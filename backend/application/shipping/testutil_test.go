package shipping_test

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy n
// separate tenant.WithTenant Begin/set_config/Commit sequences -- same
// pattern as application/channels' own test helper.
func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

// --- MyParcelClient/MyParcelClientFactory fakes ---

type fakeMyParcelClient struct {
	createShipmentFn  func(ctx context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error)
	getLabelFn        func(ctx context.Context, myParcelShipmentID int64, format string) (integrationsDomain.LabelResult, error)
	getTrackTraceFn   func(ctx context.Context, myParcelShipmentID int64) (integrationsDomain.TrackTraceResult, error)
	registerWebhookFn func(ctx context.Context, callbackURL string) error
}

func (f *fakeMyParcelClient) CreateShipment(ctx context.Context, in integrationsDomain.CreateShipmentInput) (integrationsDomain.ShipmentResult, error) {
	return f.createShipmentFn(ctx, in)
}
func (f *fakeMyParcelClient) GetLabel(ctx context.Context, id int64, format string) (integrationsDomain.LabelResult, error) {
	return f.getLabelFn(ctx, id, format)
}
func (f *fakeMyParcelClient) GetTrackTrace(ctx context.Context, id int64) (integrationsDomain.TrackTraceResult, error) {
	return f.getTrackTraceFn(ctx, id)
}
func (f *fakeMyParcelClient) RegisterWebhook(ctx context.Context, callbackURL string) error {
	return f.registerWebhookFn(ctx, callbackURL)
}

type fakeMyParcelClientFactory struct {
	client       integrationsDomain.MyParcelClient
	newClientErr error
}

func (f *fakeMyParcelClientFactory) NewClient(apiKey string) (integrationsDomain.MyParcelClient, error) {
	if f.newClientErr != nil {
		return nil, f.newClientErr
	}
	return f.client, nil
}

// --- ShipmentRepository fake ---

type fakeShipmentRepo struct {
	createFn                   func(ctx context.Context, s *shippingDomain.Shipment) error
	findByIDFn                 func(ctx context.Context, id string) (*shippingDomain.Shipment, error)
	findByMyParcelShipmentIDFn func(ctx context.Context, id int64) (*shippingDomain.Shipment, error)
	findBySalesOrderIDFn       func(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error)
	findStaleNonTerminalFn     func(ctx context.Context, olderThan time.Time, limit int) ([]shippingDomain.Shipment, error)
	updateStatusFn             func(ctx context.Context, id string, status int, statusMain string) error
}

func (f *fakeShipmentRepo) Create(ctx context.Context, s *shippingDomain.Shipment) error {
	return f.createFn(ctx, s)
}
func (f *fakeShipmentRepo) FindByID(ctx context.Context, id string) (*shippingDomain.Shipment, error) {
	return f.findByIDFn(ctx, id)
}
func (f *fakeShipmentRepo) FindByMyParcelShipmentID(ctx context.Context, id int64) (*shippingDomain.Shipment, error) {
	return f.findByMyParcelShipmentIDFn(ctx, id)
}
func (f *fakeShipmentRepo) FindBySalesOrderID(ctx context.Context, salesOrderID string) ([]shippingDomain.Shipment, error) {
	return f.findBySalesOrderIDFn(ctx, salesOrderID)
}
func (f *fakeShipmentRepo) FindStaleNonTerminal(ctx context.Context, olderThan time.Time, limit int) ([]shippingDomain.Shipment, error) {
	return f.findStaleNonTerminalFn(ctx, olderThan, limit)
}
func (f *fakeShipmentRepo) UpdateStatus(ctx context.Context, id string, status int, statusMain string) error {
	return f.updateStatusFn(ctx, id, status, statusMain)
}

// --- ShipmentStatusEventRepository fake ---

type fakeShipmentStatusEventRepo struct {
	appendFn func(ctx context.Context, e *shippingDomain.ShipmentStatusEvent) error
}

func (f *fakeShipmentStatusEventRepo) Append(ctx context.Context, e *shippingDomain.ShipmentStatusEvent) error {
	return f.appendFn(ctx, e)
}

// --- ShippingCarrierOverrideRepository fake ---

type fakeShippingCarrierOverrideRepo struct {
	listByConnectionFn           func(ctx context.Context, connectionID string) ([]shippingDomain.ShippingCarrierOverride, error)
	findByConnectionAndCarrierFn func(ctx context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error)
	upsertFn                     func(ctx context.Context, o *shippingDomain.ShippingCarrierOverride) error
}

func (f *fakeShippingCarrierOverrideRepo) ListByConnection(ctx context.Context, connectionID string) ([]shippingDomain.ShippingCarrierOverride, error) {
	return f.listByConnectionFn(ctx, connectionID)
}
func (f *fakeShippingCarrierOverrideRepo) FindByConnectionAndCarrier(ctx context.Context, connectionID string, carrier shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
	return f.findByConnectionAndCarrierFn(ctx, connectionID, carrier)
}
func (f *fakeShippingCarrierOverrideRepo) Upsert(ctx context.Context, o *shippingDomain.ShippingCarrierOverride) error {
	return f.upsertFn(ctx, o)
}
