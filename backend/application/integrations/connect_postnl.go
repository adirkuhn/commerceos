package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// postnlTrackingURLTemplate is PostNL's public track & trace page (see
// shipping.BuildCarrierTrackingURL for the placeholders). Domestic only:
// the template has no country placeholder.
const postnlTrackingURLTemplate = "https://jouw.postnl.nl/track-and-trace/{barcode}-NL-{postcode}"

// postnlStatusPollMinutes: PostNL sends no webhooks and asks callers not
// to poll its status API more than needed, so shipments are re-checked
// hourly.
const postnlStatusPollMinutes = 60

// postnlAccount normalizes a PostNL customer code/number, within the
// lengths PostNL's Shipment v4 API accepts.
func postnlAccount(customerCode, customerNumber string) (string, string, error) {
	code := strings.ToUpper(strings.TrimSpace(customerCode))
	number := strings.TrimSpace(customerNumber)
	if code == "" || len(code) > 4 || number == "" || len(number) > 10 {
		return "", "", integrationsDomain.ErrPostNLAccountInvalid
	}
	return code, number, nil
}

type ConnectPostNLInput struct {
	APIKey         string
	CustomerCode   string
	CustomerNumber string
	Sandbox        bool
}

type carrierOverrideUpserter interface {
	Upsert(ctx context.Context, o *shippingDomain.ShippingCarrierOverride) error
}

type syncScheduleUpserter interface {
	Upsert(ctx context.Context, s *channelsDomain.SyncSchedule) error
}

// ConnectPostNLUseCase connects a PostNL business account directly (not
// through MyParcel). Like ConnectMyParcelUseCase it seeds the connection's
// shipping settings; it also sets PostNL's tracking link and turns on the
// shipment status polling, since PostNL has no webhooks.
type ConnectPostNLUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	settingsRepo     shippingDomain.ShippingConnectionSettingsRepository
	overrides        carrierOverrideUpserter
	schedules        syncScheduleUpserter
	outboxRepo       outbox.Repository
	encryptionSecret string
}

func NewConnectPostNLUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	settingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	overrides carrierOverrideUpserter,
	schedules syncScheduleUpserter,
	outboxRepo outbox.Repository,
	encryptionSecret string,
) *ConnectPostNLUseCase {
	return &ConnectPostNLUseCase{
		db: db, connRepo: connRepo, settingsRepo: settingsRepo, overrides: overrides, schedules: schedules,
		outboxRepo: outboxRepo, encryptionSecret: encryptionSecret,
	}
}

func (uc *ConnectPostNLUseCase) Execute(ctx context.Context, in ConnectPostNLInput) (*integrationsDomain.Connection, error) {
	tenantID := tenant.IDFromContext(ctx)
	if tenantID == "" {
		return nil, fmt.Errorf("connectPostNL: %w", integrationsDomain.ErrNoTenantContext)
	}
	apiKey := strings.TrimSpace(in.APIKey)
	code, number, err := postnlAccount(in.CustomerCode, in.CustomerNumber)
	if err != nil || apiKey == "" {
		return nil, fmt.Errorf("connectPostNL: %w", integrationsDomain.ErrPostNLAccountInvalid)
	}
	encryptedKey, err := crypto.Encrypt(uc.encryptionSecret, apiKey)
	if err != nil {
		return nil, fmt.Errorf("connectPostNL: %w", err)
	}

	var conn integrationsDomain.Connection
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		conn = integrationsDomain.Connection{
			TenantID: tenantID, Provider: integrationsDomain.ProviderPostNL, ConsumerSecretEncrypted: encryptedKey,
			PostNLCustomerCode: code, PostNLCustomerNumber: number, PostNLSandbox: in.Sandbox,
			Status: integrationsDomain.StatusConnected, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := integrationsDomain.SaveConnection(ctx, uc.connRepo, &conn); err != nil {
			return err
		}
		if err := uc.settingsRepo.Upsert(ctx, &shippingDomain.ShippingConnectionSettings{
			ConnectionID: conn.ID, TenantID: tenantID,
			DefaultCarrier: shippingDomain.CarrierPostNL, DefaultPackageType: shippingDomain.PackageTypePackage,
			DefaultLabelFormat: shippingDomain.LabelFormatPDF, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		tmpl := postnlTrackingURLTemplate
		if err := uc.overrides.Upsert(ctx, &shippingDomain.ShippingCarrierOverride{
			TenantID: tenantID, ConnectionID: conn.ID, Carrier: shippingDomain.CarrierPostNL,
			TrackingURLTemplate: &tmpl, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		next := now.Add(postnlStatusPollMinutes * time.Minute)
		if err := uc.schedules.Upsert(ctx, &channelsDomain.SyncSchedule{
			TenantID: tenantID, ConnectionID: conn.ID,
			EntityType: channelsDomain.SyncEntityShipmentReconciliation, Direction: channelsDomain.SyncDirectionImport,
			Enabled: true, IntervalMinutes: postnlStatusPollMinutes, NextRunAt: &next, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
		event := events.IntegrationConnected{
			Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: now},
			TenantID: tenantID, Provider: string(integrationsDomain.ProviderPostNL), ConnectionID: conn.ID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outboxRepo.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("connectPostNL: %w", err)
	}
	return &conn, nil
}
