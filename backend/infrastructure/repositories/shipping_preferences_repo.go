package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type shippingPreferencesModel struct {
	TenantID            string  `gorm:"primaryKey;type:uuid"`
	DefaultConnectionID *string `gorm:"type:uuid"`
	SenderAddress       []byte  `gorm:"type:jsonb"`
	ReturnAddress       []byte  `gorm:"type:jsonb"`
	UpdatedAt           time.Time
}

// shippingAddressJSON is ShippingAddress's stored shape.
type shippingAddressJSON struct {
	Company      string `json:"company"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	NumberSuffix string `json:"number_suffix,omitempty"`
	PostalCode   string `json:"postal_code"`
	City         string `json:"city"`
	CC           string `json:"cc"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

func encodeShippingAddress(a *shippingDomain.ShippingAddress) ([]byte, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(shippingAddressJSON(*a))
}

func decodeShippingAddress(raw []byte) (*shippingDomain.ShippingAddress, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var j shippingAddressJSON
	if err := json.Unmarshal(raw, &j); err != nil {
		return nil, err
	}
	a := shippingDomain.ShippingAddress(j)
	return &a, nil
}

func (shippingPreferencesModel) TableName() string { return "shipping_preferences" }

type ShippingPreferencesRepo struct{ db *gorm.DB }

var _ shippingDomain.ShippingPreferencesRepository = (*ShippingPreferencesRepo)(nil)

func NewShippingPreferencesRepo(db *gorm.DB) *ShippingPreferencesRepo {
	return &ShippingPreferencesRepo{db: db}
}

func (r *ShippingPreferencesRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("shippingPreferencesRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ShippingPreferencesRepo) Get(ctx context.Context) (*shippingDomain.ShippingPreferences, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m shippingPreferencesModel
	err = tx.First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("shippingPreferencesRepo.Get: %w", err)
	}
	sender, err := decodeShippingAddress(m.SenderAddress)
	if err != nil {
		return nil, fmt.Errorf("shippingPreferencesRepo.Get: sender address: %w", err)
	}
	ret, err := decodeShippingAddress(m.ReturnAddress)
	if err != nil {
		return nil, fmt.Errorf("shippingPreferencesRepo.Get: return address: %w", err)
	}
	return &shippingDomain.ShippingPreferences{
		TenantID: m.TenantID, DefaultConnectionID: m.DefaultConnectionID, SenderAddress: sender, ReturnAddress: ret, UpdatedAt: m.UpdatedAt,
	}, nil
}

func (r *ShippingPreferencesRepo) Upsert(ctx context.Context, p *shippingDomain.ShippingPreferences) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	sender, err := encodeShippingAddress(p.SenderAddress)
	if err != nil {
		return fmt.Errorf("shippingPreferencesRepo.Upsert: sender address: %w", err)
	}
	ret, err := encodeShippingAddress(p.ReturnAddress)
	if err != nil {
		return fmt.Errorf("shippingPreferencesRepo.Upsert: return address: %w", err)
	}
	m := shippingPreferencesModel{TenantID: p.TenantID, DefaultConnectionID: p.DefaultConnectionID, SenderAddress: sender, ReturnAddress: ret, UpdatedAt: p.UpdatedAt}
	err = tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"default_connection_id", "sender_address", "return_address", "updated_at"}),
	}).Create(&m).Error
	if err != nil {
		return fmt.Errorf("shippingPreferencesRepo.Upsert: %w", err)
	}
	return nil
}
