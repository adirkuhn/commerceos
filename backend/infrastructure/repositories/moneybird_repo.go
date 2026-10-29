package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

func moneybirdTx(ctx context.Context, repo string) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("%s: operation called outside a transaction", repo)
	}
	return tx.WithContext(ctx), nil
}

// ── settings ─────────────────────────────────────────────────────────────────

type moneybirdTaxRateJSON struct {
	RateBasisPoints    int32  `json:"rate_basis_points"`
	MoneybirdTaxRateID string `json:"moneybird_tax_rate_id"`
}

type moneybirdPaymentMethodJSON struct {
	Method                 string `json:"method"`
	MoneybirdPaymentMethod string `json:"moneybird_payment_method"`
}

type moneybirdSettingsModel struct {
	ConnectionID           string                              `gorm:"primaryKey;type:uuid"`
	TenantID               string                              `gorm:"not null;type:uuid"`
	AdministrationID       string                              `gorm:"not null"`
	AdministrationName     string                              `gorm:"not null"`
	DefaultLedgerAccountID string                              `gorm:"column:default_ledger_account_id;not null"`
	GuestContactID         string                              `gorm:"not null"`
	AutoSync               bool                                `gorm:"not null"`
	TaxRates               jsonb[[]moneybirdTaxRateJSON]       `gorm:"type:jsonb"`
	PaymentMethods         jsonb[[]moneybirdPaymentMethodJSON] `gorm:"type:jsonb"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (moneybirdSettingsModel) TableName() string { return "moneybird_settings" }

type MoneybirdSettingsRepo struct{ db *gorm.DB }

var _ mb.SettingsRepository = (*MoneybirdSettingsRepo)(nil)

func NewMoneybirdSettingsRepo(db *gorm.DB) *MoneybirdSettingsRepo { return &MoneybirdSettingsRepo{db: db} }

func (r *MoneybirdSettingsRepo) Get(ctx context.Context, connectionID string) (*mb.Settings, error) {
	tx, err := moneybirdTx(ctx, "moneybirdSettingsRepo")
	if err != nil {
		return nil, err
	}
	var m moneybirdSettingsModel
	err = tx.Where("connection_id = ?", connectionID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("moneybirdSettingsRepo.Get: %w", err)
	}
	s := &mb.Settings{
		ConnectionID: m.ConnectionID, TenantID: m.TenantID, AdministrationID: m.AdministrationID,
		AdministrationName: m.AdministrationName, DefaultLedgerAccountID: m.DefaultLedgerAccountID,
		GuestContactID: m.GuestContactID, AutoSync: m.AutoSync, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
	for _, t := range m.TaxRates.V {
		s.TaxRates = append(s.TaxRates, mb.TaxRateMapping{RateBasisPoints: t.RateBasisPoints, MoneybirdTaxRateID: t.MoneybirdTaxRateID})
	}
	for _, p := range m.PaymentMethods.V {
		s.PaymentMethods = append(s.PaymentMethods, mb.PaymentMethodMapping{Method: p.Method, MoneybirdPaymentMethod: p.MoneybirdPaymentMethod})
	}
	return s, nil
}

func (r *MoneybirdSettingsRepo) Upsert(ctx context.Context, s *mb.Settings) error {
	tx, err := moneybirdTx(ctx, "moneybirdSettingsRepo")
	if err != nil {
		return err
	}
	taxRates := make([]moneybirdTaxRateJSON, 0, len(s.TaxRates))
	for _, t := range s.TaxRates {
		taxRates = append(taxRates, moneybirdTaxRateJSON{RateBasisPoints: t.RateBasisPoints, MoneybirdTaxRateID: t.MoneybirdTaxRateID})
	}
	methods := make([]moneybirdPaymentMethodJSON, 0, len(s.PaymentMethods))
	for _, p := range s.PaymentMethods {
		methods = append(methods, moneybirdPaymentMethodJSON{Method: p.Method, MoneybirdPaymentMethod: p.MoneybirdPaymentMethod})
	}
	m := moneybirdSettingsModel{
		ConnectionID: s.ConnectionID, TenantID: s.TenantID, AdministrationID: s.AdministrationID,
		AdministrationName: s.AdministrationName, DefaultLedgerAccountID: s.DefaultLedgerAccountID,
		GuestContactID: s.GuestContactID, AutoSync: s.AutoSync,
		TaxRates: jsonb[[]moneybirdTaxRateJSON]{V: taxRates}, PaymentMethods: jsonb[[]moneybirdPaymentMethodJSON]{V: methods},
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "connection_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"administration_id", "administration_name", "default_ledger_account_id", "guest_contact_id",
			"auto_sync", "tax_rates", "payment_methods", "updated_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("moneybirdSettingsRepo.Upsert: %w", err)
	}
	return nil
}

// ── documents ────────────────────────────────────────────────────────────────

type moneybirdDocumentModel struct {
	ID                string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID          string `gorm:"not null;type:uuid"`
	ConnectionID      string `gorm:"not null;type:uuid"`
	SourceType        string `gorm:"not null"`
	SourceID          string `gorm:"not null;type:uuid"`
	MoneybirdID       string `gorm:"not null"`
	Reference         string `gorm:"not null"`
	Status            string `gorm:"not null"`
	PaymentRegistered bool   `gorm:"not null"`
	Error             string `gorm:"not null"`
	SyncedAt          time.Time
}

func (moneybirdDocumentModel) TableName() string { return "moneybird_documents" }

type MoneybirdDocumentRepo struct{ db *gorm.DB }

var _ mb.DocumentRepository = (*MoneybirdDocumentRepo)(nil)

func NewMoneybirdDocumentRepo(db *gorm.DB) *MoneybirdDocumentRepo { return &MoneybirdDocumentRepo{db: db} }

func (r *MoneybirdDocumentRepo) Find(ctx context.Context, sourceType mb.SourceType, sourceID string) (*mb.Document, error) {
	tx, err := moneybirdTx(ctx, "moneybirdDocumentRepo")
	if err != nil {
		return nil, err
	}
	var m moneybirdDocumentModel
	err = tx.Where("source_type = ? AND source_id = ?", string(sourceType), sourceID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("moneybirdDocumentRepo.Find: %w", err)
	}
	return &mb.Document{
		ID: m.ID, TenantID: m.TenantID, ConnectionID: m.ConnectionID, SourceType: mb.SourceType(m.SourceType),
		SourceID: m.SourceID, MoneybirdID: m.MoneybirdID, Reference: m.Reference, Status: mb.DocumentStatus(m.Status),
		PaymentRegistered: m.PaymentRegistered, Error: m.Error, SyncedAt: m.SyncedAt,
	}, nil
}

func (r *MoneybirdDocumentRepo) Save(ctx context.Context, d *mb.Document) error {
	tx, err := moneybirdTx(ctx, "moneybirdDocumentRepo")
	if err != nil {
		return err
	}
	m := moneybirdDocumentModel{
		TenantID: d.TenantID, ConnectionID: d.ConnectionID, SourceType: string(d.SourceType), SourceID: d.SourceID,
		MoneybirdID: d.MoneybirdID, Reference: d.Reference, Status: string(d.Status),
		PaymentRegistered: d.PaymentRegistered, Error: d.Error, SyncedAt: d.SyncedAt,
	}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "source_type"}, {Name: "source_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"connection_id", "moneybird_id", "reference", "status", "payment_registered", "error", "synced_at",
		}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("moneybirdDocumentRepo.Save: %w", err)
	}
	d.ID = m.ID
	return nil
}

// ── contacts ─────────────────────────────────────────────────────────────────

type moneybirdContactModel struct {
	TenantID           string `gorm:"primaryKey;type:uuid"`
	ConnectionID       string `gorm:"primaryKey;type:uuid"`
	CustomerID         string `gorm:"primaryKey;type:uuid"`
	MoneybirdContactID string `gorm:"not null"`
	CreatedAt          time.Time
}

func (moneybirdContactModel) TableName() string { return "moneybird_contacts" }

type MoneybirdContactRepo struct{ db *gorm.DB }

var _ mb.ContactRepository = (*MoneybirdContactRepo)(nil)

func NewMoneybirdContactRepo(db *gorm.DB) *MoneybirdContactRepo { return &MoneybirdContactRepo{db: db} }

func (r *MoneybirdContactRepo) Find(ctx context.Context, connectionID, customerID string) (string, error) {
	tx, err := moneybirdTx(ctx, "moneybirdContactRepo")
	if err != nil {
		return "", err
	}
	var m moneybirdContactModel
	err = tx.Where("connection_id = ? AND customer_id = ?", connectionID, customerID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("moneybirdContactRepo.Find: %w", err)
	}
	return m.MoneybirdContactID, nil
}

func (r *MoneybirdContactRepo) Save(ctx context.Context, tenantID, connectionID, customerID, moneybirdContactID string) error {
	tx, err := moneybirdTx(ctx, "moneybirdContactRepo")
	if err != nil {
		return err
	}
	m := moneybirdContactModel{TenantID: tenantID, ConnectionID: connectionID, CustomerID: customerID, MoneybirdContactID: moneybirdContactID, CreatedAt: time.Now()}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "connection_id"}, {Name: "customer_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"moneybird_contact_id"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("moneybirdContactRepo.Save: %w", err)
	}
	return nil
}
