package pms

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// productTranslationsRepo is ProductRepository narrowed to what setting
// translations needs, plus the translations-only write.
type productTranslationsRepo interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
	SetTranslations(ctx context.Context, productID string, tr map[locale.Locale]pmsDomain.ProductTexts) error
}

// SetProductTranslationsUseCase replaces a product's name/description in
// the store's other languages. Separate from UpdateProductUseCase so the
// many product updates (admin form, channel imports) never touch them.
type SetProductTranslationsUseCase struct {
	db     *gorm.DB
	repo   productTranslationsRepo
	outbox outbox.Repository
}

func NewSetProductTranslationsUseCase(db *gorm.DB, repo productTranslationsRepo, outboxRepo outbox.Repository) *SetProductTranslationsUseCase {
	return &SetProductTranslationsUseCase{db: db, repo: repo, outbox: outboxRepo}
}

func (uc *SetProductTranslationsUseCase) Execute(ctx context.Context, productID string, tr map[locale.Locale]pmsDomain.ProductTexts) (*pmsDomain.Product, error) {
	clean, err := pmsDomain.ValidProductTranslations(tr)
	if err != nil {
		return nil, fmt.Errorf("setProductTranslations: %w", err)
	}
	var product *pmsDomain.Product
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.SetTranslations(ctx, productID, clean); err != nil {
			return err
		}
		if product, err = uc.repo.FindByID(ctx, productID); err != nil {
			return err
		}
		event := events.ProductTranslated{
			Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now()},
			TenantID: tenant.IDFromContext(ctx), ProductID: productID,
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		return uc.outbox.Enqueue(ctx, event.EventName(), event.EventID, payload, event.OccurredAt)
	})
	if err != nil {
		return nil, fmt.Errorf("setProductTranslations: %w", err)
	}
	return product, nil
}
