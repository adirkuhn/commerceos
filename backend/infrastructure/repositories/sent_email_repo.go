package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/internal/txctx"
)

type sentEmailModel struct {
	ID             string    `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	TenantID       string    `gorm:"column:tenant_id"`
	Source         string    `gorm:"column:source"`
	Recipient      string    `gorm:"column:recipient"`
	Subject        string    `gorm:"column:subject"`
	HTMLBody       string    `gorm:"column:html_body"`
	Status         string    `gorm:"column:status"`
	Reason         string    `gorm:"column:reason"`
	RuleID         *string   `gorm:"column:rule_id"`
	TemplateID     *string   `gorm:"column:template_id"`
	TriggerSubject string    `gorm:"column:trigger_subject"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (sentEmailModel) TableName() string { return "sent_emails" }

// SentEmailRepo implements mcDomain.SentEmailRepository.
type SentEmailRepo struct {
	db *gorm.DB
}

func NewSentEmailRepo(db *gorm.DB) *SentEmailRepo { return &SentEmailRepo{db: db} }

var _ mcDomain.SentEmailRepository = (*SentEmailRepo)(nil)

func (r *SentEmailRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := txctx.FromContext(ctx)
	if !ok {
		return nil, errors.New("sentEmailRepo: operation called outside a transaction")
	}
	return tx.WithContext(ctx), nil
}

func (r *SentEmailRepo) Create(ctx context.Context, e *mcDomain.SentEmail) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	m := sentEmailModel{
		TenantID: e.TenantID, Source: string(e.Source), Recipient: e.Recipient, Subject: e.Subject, HTMLBody: e.HTMLBody,
		Status: string(e.Status), Reason: e.Reason, RuleID: optionalID(e.RuleID), TemplateID: optionalID(e.TemplateID),
		TriggerSubject: e.TriggerSubject, CreatedAt: e.CreatedAt,
	}
	if err := tx.Create(&m).Error; err != nil {
		return fmt.Errorf("sentEmailRepo.Create: %w", err)
	}
	e.ID = m.ID
	return nil
}

func (r *SentEmailRepo) List(ctx context.Context, f mcDomain.SentEmailFilter) ([]mcDomain.SentEmail, int, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	f = f.Normalized()
	q := tx.Model(&sentEmailModel{})
	if f.Status != "" {
		q = q.Where("status = ?", string(f.Status))
	}
	if f.Source != "" {
		q = q.Where("source = ?", string(f.Source))
	}
	if s := strings.TrimSpace(f.Query); s != "" {
		like := "%" + escapeLike(s) + "%"
		q = q.Where("(recipient ILIKE ? OR subject ILIKE ?)", like, like)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at < ?", *f.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("sentEmailRepo.List: %w", err)
	}
	var rows []sentEmailModel
	err = q.Omit("html_body").Order("created_at DESC, id DESC").
		Limit(f.PerPage).Offset((f.Page - 1) * f.PerPage).Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("sentEmailRepo.List: %w", err)
	}
	out := make([]mcDomain.SentEmail, len(rows))
	for i, m := range rows {
		out[i] = toSentEmailDomain(m)
	}
	return out, int(total), nil
}

func (r *SentEmailRepo) FindByID(ctx context.Context, id string) (*mcDomain.SentEmail, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var m sentEmailModel
	if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mcDomain.ErrSentEmailNotFound
		}
		return nil, fmt.Errorf("sentEmailRepo.FindByID: %w", err)
	}
	e := toSentEmailDomain(m)
	return &e, nil
}

func (r *SentEmailRepo) Delete(ctx context.Context, ids []string) ([]string, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	var deleted []sentEmailModel
	if err := tx.Raw(`DELETE FROM sent_emails WHERE id IN ? RETURNING id`, ids).Scan(&deleted).Error; err != nil {
		return nil, fmt.Errorf("sentEmailRepo.Delete: %w", err)
	}
	out := make([]string, len(deleted))
	for i, m := range deleted {
		out[i] = m.ID
	}
	return out, nil
}

func toSentEmailDomain(m sentEmailModel) mcDomain.SentEmail {
	return mcDomain.SentEmail{
		ID: m.ID, TenantID: m.TenantID, Source: mcDomain.SentEmailSource(m.Source), Recipient: m.Recipient, Subject: m.Subject,
		HTMLBody: m.HTMLBody, Status: mcDomain.SentEmailStatus(m.Status), Reason: m.Reason,
		RuleID: derefID(m.RuleID), TemplateID: derefID(m.TemplateID), TriggerSubject: m.TriggerSubject, CreatedAt: m.CreatedAt,
	}
}

func optionalID(id string) *string {
	if id == "" {
		return nil
	}
	return &id
}

func derefID(id *string) string {
	if id == nil {
		return ""
	}
	return *id
}

// escapeLike makes a search term match literally inside ILIKE '%...%'
// (backslash is Postgres' default LIKE escape).
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
