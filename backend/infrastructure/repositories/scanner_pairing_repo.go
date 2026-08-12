package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type scannerPairingCodeModel struct {
	ID        string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  string `gorm:"not null;type:uuid"`
	Code      string `gorm:"not null"`
	Token     string `gorm:"not null"`
	UserID    string `gorm:"not null"`
	Role      string `gorm:"not null"`
	Status    string `gorm:"not null;default:pending"`
	ExpiresAt time.Time
	CreatedAt time.Time
	ClaimedAt *time.Time
}

func (scannerPairingCodeModel) TableName() string { return "scanner_pairing_codes" }

type ScannerPairingCodeRepo struct{ db *gorm.DB }

func NewScannerPairingCodeRepo(db *gorm.DB) *ScannerPairingCodeRepo {
	return &ScannerPairingCodeRepo{db: db}
}

func (r *ScannerPairingCodeRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("scannerPairingCodeRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *ScannerPairingCodeRepo) Create(ctx context.Context, p *iamDomain.ScannerPairingCode) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := scannerPairingCodeModel{
		ID: p.ID, TenantID: p.TenantID, Code: p.Code, Token: p.Token, UserID: p.UserID, Role: p.Role,
		Status: string(p.Status), ExpiresAt: p.ExpiresAt, CreatedAt: p.CreatedAt, ClaimedAt: p.ClaimedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("scannerPairingCodeRepo.Create: %w", err)
	}
	p.ID = m.ID
	return nil
}

func (r *ScannerPairingCodeRepo) FindByID(ctx context.Context, id string) (*iamDomain.ScannerPairingCode, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m scannerPairingCodeModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, iamDomain.ErrPairingCodeNotFound
		}
		return nil, fmt.Errorf("scannerPairingCodeRepo.FindByID: %w", err)
	}
	return toScannerPairingCodeDomain(m), nil
}

func (r *ScannerPairingCodeRepo) FindPendingByCode(ctx context.Context, code string) (*iamDomain.ScannerPairingCode, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m scannerPairingCodeModel
	err = tx.Where("code = ? AND status = ? AND expires_at > ?", code, string(iamDomain.ScannerPairingCodeStatusPending), time.Now()).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("scannerPairingCodeRepo.FindPendingByCode: %w", err)
	}
	return toScannerPairingCodeDomain(m), nil
}

func (r *ScannerPairingCodeRepo) FindPendingByToken(ctx context.Context, token string) (*iamDomain.ScannerPairingCode, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m scannerPairingCodeModel
	err = tx.Where("token = ? AND status = ? AND expires_at > ?", token, string(iamDomain.ScannerPairingCodeStatusPending), time.Now()).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("scannerPairingCodeRepo.FindPendingByToken: %w", err)
	}
	return toScannerPairingCodeDomain(m), nil
}

func (r *ScannerPairingCodeRepo) MarkClaimed(ctx context.Context, id string, claimedAt time.Time) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Model(&scannerPairingCodeModel{}).Where("id = ?", id).
		Updates(map[string]any{"status": string(iamDomain.ScannerPairingCodeStatusClaimed), "claimed_at": claimedAt})
	if res.Error != nil {
		return fmt.Errorf("scannerPairingCodeRepo.MarkClaimed: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return iamDomain.ErrPairingCodeNotFound
	}
	return nil
}

func toScannerPairingCodeDomain(m scannerPairingCodeModel) *iamDomain.ScannerPairingCode {
	return &iamDomain.ScannerPairingCode{
		ID: m.ID, TenantID: m.TenantID, Code: m.Code, Token: m.Token, UserID: m.UserID, Role: m.Role,
		Status: iamDomain.ScannerPairingCodeStatus(m.Status), ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt, ClaimedAt: m.ClaimedAt,
	}
}
