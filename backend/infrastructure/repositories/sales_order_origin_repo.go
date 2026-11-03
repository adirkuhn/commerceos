package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type salesOrderOriginModel struct {
	ID              string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID        string  `gorm:"not null;type:uuid"`
	SalesOrderID    string  `gorm:"not null;type:uuid"`
	Origin          string  `gorm:"not null"`
	ConnectionID    *string `gorm:"type:uuid"`
	ExternalOrderID *string
	ExternalKey     *string
	CreatedAt       time.Time
}

func (salesOrderOriginModel) TableName() string { return "sales_order_origins" }

// SalesOrderOriginRepo implements integrationsDomain.SalesOrderOriginRepository.
type SalesOrderOriginRepo struct{ db *gorm.DB }

func NewSalesOrderOriginRepo(db *gorm.DB) *SalesOrderOriginRepo { return &SalesOrderOriginRepo{db: db} }

func (r *SalesOrderOriginRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("salesOrderOrigins: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

// Save upserts on sales_order_id: one row per order. Migration 246's
// trigger may already have written the row from the old sales_orders
// columns in the same transaction; what the application saves wins.
func (r *SalesOrderOriginRepo) Save(ctx context.Context, o *integrationsDomain.SalesOrderOrigin) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toSalesOrderOriginModel(*o)
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "sales_order_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"origin", "connection_id", "external_order_id", "external_key"}),
	}).Create(&m).Error
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("salesOrderOrigins.Save: %s: %w", pgErr.ConstraintName, integrationsDomain.ErrExternalOrderAlreadyImported)
	}
	if err != nil {
		return fmt.Errorf("salesOrderOrigins.Save: %w", err)
	}
	// On conflict Postgres returns the existing row's id.
	o.ID = m.ID
	return nil
}

func (r *SalesOrderOriginRepo) FindBySalesOrderID(ctx context.Context, salesOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	return r.findOne(ctx, "FindBySalesOrderID", "sales_order_id = ?", salesOrderID)
}

func (r *SalesOrderOriginRepo) FindByConnectionExternalID(ctx context.Context, connectionID, externalOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	return r.findOne(ctx, "FindByConnectionExternalID", "connection_id = ? AND external_order_id = ?", connectionID, externalOrderID)
}

func (r *SalesOrderOriginRepo) FindByOriginExternalID(ctx context.Context, origin integrationsDomain.Origin, externalOrderID string) (*integrationsDomain.SalesOrderOrigin, error) {
	return r.findOne(ctx, "FindByOriginExternalID", "origin = ? AND connection_id IS NULL AND external_order_id = ?", string(origin), externalOrderID)
}

func (r *SalesOrderOriginRepo) findOne(ctx context.Context, op, where string, args ...any) (*integrationsDomain.SalesOrderOrigin, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m salesOrderOriginModel
	err = tx.Where(where, args...).Take(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, integrationsDomain.ErrSalesOrderOriginNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("salesOrderOrigins.%s: %w", op, err)
	}
	o := fromSalesOrderOriginModel(m)
	return &o, nil
}

func toSalesOrderOriginModel(o integrationsDomain.SalesOrderOrigin) salesOrderOriginModel {
	return salesOrderOriginModel{
		ID: o.ID, TenantID: o.TenantID, SalesOrderID: o.SalesOrderID, Origin: string(o.Origin), ConnectionID: o.ConnectionID,
		ExternalOrderID: nilIfEmpty(o.ExternalOrderID), ExternalKey: nilIfEmpty(o.ExternalKey), CreatedAt: o.CreatedAt,
	}
}

func fromSalesOrderOriginModel(m salesOrderOriginModel) integrationsDomain.SalesOrderOrigin {
	return integrationsDomain.SalesOrderOrigin{
		ID: m.ID, TenantID: m.TenantID, SalesOrderID: m.SalesOrderID, Origin: integrationsDomain.Origin(m.Origin),
		ConnectionID: m.ConnectionID, ExternalOrderID: derefString(m.ExternalOrderID), ExternalKey: derefString(m.ExternalKey), CreatedAt: m.CreatedAt,
	}
}

// nilIfEmpty / derefString map the domain's "" (no reference) onto NULL,
// so the partial unique indexes only ever see real external ids.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
