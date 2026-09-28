package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type newsletterSubscriberModel struct {
	ID                 string  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID           string  `gorm:"type:uuid"`
	SalesChannelID     *string `gorm:"type:uuid"`
	Email              string
	Status             string
	ConfirmTokenHash   *string
	UnsubscribeToken   string
	ConsentAt          time.Time
	ConfirmationSentAt time.Time
	ConfirmedAt        *time.Time
	UnsubscribedAt     *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (newsletterSubscriberModel) TableName() string { return "newsletter_subscribers" }

func toSubscriberModel(s newsletter.Subscriber) newsletterSubscriberModel {
	m := newsletterSubscriberModel{
		ID: s.ID, TenantID: s.TenantID, Email: s.Email, Status: string(s.Status), UnsubscribeToken: s.UnsubscribeToken,
		ConsentAt: s.ConsentAt, ConfirmationSentAt: s.ConfirmationSentAt, ConfirmedAt: s.ConfirmedAt,
		UnsubscribedAt: s.UnsubscribedAt, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
	if s.SalesChannelID != "" {
		ch := s.SalesChannelID
		m.SalesChannelID = &ch
	}
	if s.ConfirmTokenHash != "" {
		h := s.ConfirmTokenHash
		m.ConfirmTokenHash = &h
	}
	return m
}

func (m newsletterSubscriberModel) toDomain() newsletter.Subscriber {
	s := newsletter.Subscriber{
		ID: m.ID, TenantID: m.TenantID, Email: m.Email, Status: newsletter.Status(m.Status), UnsubscribeToken: m.UnsubscribeToken,
		ConsentAt: m.ConsentAt, ConfirmationSentAt: m.ConfirmationSentAt, ConfirmedAt: m.ConfirmedAt,
		UnsubscribedAt: m.UnsubscribedAt, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
	if m.SalesChannelID != nil {
		s.SalesChannelID = *m.SalesChannelID
	}
	if m.ConfirmTokenHash != nil {
		s.ConfirmTokenHash = *m.ConfirmTokenHash
	}
	return s
}

type NewsletterRepo struct{ db *gorm.DB }

func NewNewsletterRepo(db *gorm.DB) *NewsletterRepo { return &NewsletterRepo{db: db} }

var _ newsletter.Repository = (*NewsletterRepo)(nil)

func (r *NewsletterRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("newsletter: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *NewsletterRepo) Create(ctx context.Context, s *newsletter.Subscriber) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toSubscriberModel(*s)
	m.ID = ""
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("newsletterRepo.Create: %w", err)
	}
	s.ID = m.ID
	return nil
}

func (r *NewsletterRepo) findOne(ctx context.Context, where string, arg string) (*newsletter.Subscriber, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var rows []newsletterSubscriberModel
	if err := tx.Where(where, arg).Limit(1).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("newsletterRepo.find: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	s := rows[0].toDomain()
	return &s, nil
}

func (r *NewsletterRepo) FindByEmail(ctx context.Context, email string) (*newsletter.Subscriber, error) {
	return r.findOne(ctx, "email = ?", email)
}

func (r *NewsletterRepo) FindByConfirmTokenHash(ctx context.Context, hash string) (*newsletter.Subscriber, error) {
	return r.findOne(ctx, "confirm_token_hash = ?", hash)
}

func (r *NewsletterRepo) FindByUnsubscribeToken(ctx context.Context, token string) (*newsletter.Subscriber, error) {
	return r.findOne(ctx, "unsubscribe_token = ?", token)
}

func (r *NewsletterRepo) Update(ctx context.Context, s *newsletter.Subscriber) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := toSubscriberModel(*s)
	if err := tx.Model(&newsletterSubscriberModel{}).Where("id = ?", s.ID).Updates(map[string]any{
		"status": m.Status, "confirm_token_hash": m.ConfirmTokenHash, "consent_at": m.ConsentAt,
		"confirmation_sent_at": m.ConfirmationSentAt, "confirmed_at": m.ConfirmedAt, "unsubscribed_at": m.UnsubscribedAt,
		"sales_channel_id": m.SalesChannelID, "updated_at": m.UpdatedAt,
	}).Error; err != nil {
		return fmt.Errorf("newsletterRepo.Update: %w", err)
	}
	return nil
}

func (r *NewsletterRepo) List(ctx context.Context, f newsletter.ListFilter) ([]newsletter.Subscriber, int64, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := tx.Model(&newsletterSubscriberModel{})
	if f.Status != nil {
		q = q.Where("status = ?", string(*f.Status))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		q = q.Where("email ILIKE ?", "%"+strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(s))+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("newsletterRepo.List count: %w", err)
	}
	var rows []newsletterSubscriberModel
	if err := q.Order("created_at DESC, id").Limit(f.Limit).Offset(f.Offset).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("newsletterRepo.List: %w", err)
	}
	out := make([]newsletter.Subscriber, len(rows))
	for i, m := range rows {
		out[i] = m.toDomain()
	}
	return out, total, nil
}

func (r *NewsletterRepo) Delete(ctx context.Context, id string) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.Where("id = ?", id).Delete(&newsletterSubscriberModel{})
	if res.Error != nil {
		return fmt.Errorf("newsletterRepo.Delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return newsletter.ErrSubscriberNotFound
	}
	return nil
}
