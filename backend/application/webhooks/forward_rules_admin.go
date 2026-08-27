package webhooks

import (
	"context"
	"fmt"
	"net/url"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// isHTTPURL guards ForwardRule.TargetURL: it becomes a real outbound HTTP
// POST (see DeliverForwardedWebhooksUseCase), so anything that isn't an
// absolute http(s) URL is rejected up front rather than failing every
// delivery attempt later.
func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// CreateForwardRuleUseCase adds one (provider, topic, target) fan-out rule
// -- the Integrations page's "add a webhook forward" action. See
// domain/webhooks.ForwardRule's doc comment: Topic == "" matches every
// event for Provider.
type CreateForwardRuleUseCase struct {
	db       *gorm.DB
	ruleRepo webhooks.ForwardRuleRepository
}

func NewCreateForwardRuleUseCase(db *gorm.DB, ruleRepo webhooks.ForwardRuleRepository) *CreateForwardRuleUseCase {
	return &CreateForwardRuleUseCase{db: db, ruleRepo: ruleRepo}
}

func (uc *CreateForwardRuleUseCase) Execute(ctx context.Context, provider, topic, targetURL string) (*webhooks.ForwardRule, error) {
	if provider == "" {
		return nil, fmt.Errorf("createForwardRule: %w", webhooks.ErrProviderRequired)
	}
	if !isHTTPURL(targetURL) {
		return nil, fmt.Errorf("createForwardRule: %w", webhooks.ErrInvalidTargetURL)
	}

	rule := webhooks.ForwardRule{
		TenantID: tenant.IDFromContext(ctx), Provider: provider, Topic: topic, TargetURL: targetURL, Active: true,
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.ruleRepo.Create(ctx, &rule)
	})
	if err != nil {
		return nil, fmt.Errorf("createForwardRule: %w", err)
	}
	return &rule, nil
}

// ListForwardRulesUseCase is the admin-facing read side of the tenant's
// configured fan-out rules.
type ListForwardRulesUseCase struct {
	db       *gorm.DB
	ruleRepo webhooks.ForwardRuleRepository
}

func NewListForwardRulesUseCase(db *gorm.DB, ruleRepo webhooks.ForwardRuleRepository) *ListForwardRulesUseCase {
	return &ListForwardRulesUseCase{db: db, ruleRepo: ruleRepo}
}

func (uc *ListForwardRulesUseCase) Execute(ctx context.Context) ([]webhooks.ForwardRule, error) {
	var out []webhooks.ForwardRule
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.ruleRepo.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listForwardRules: %w", err)
	}
	return out, nil
}

// DeleteForwardRuleUseCase removes a fan-out rule -- future events stop
// fanning out to it. Existing ForwardDelivery rows already enqueued for it
// are left alone (migration 116's ON DELETE CASCADE only fires if the row
// itself is deleted, which this never does -- a delivery already queued
// still gets its one delivery attempt drained by the worker).
type DeleteForwardRuleUseCase struct {
	db       *gorm.DB
	ruleRepo webhooks.ForwardRuleRepository
}

func NewDeleteForwardRuleUseCase(db *gorm.DB, ruleRepo webhooks.ForwardRuleRepository) *DeleteForwardRuleUseCase {
	return &DeleteForwardRuleUseCase{db: db, ruleRepo: ruleRepo}
}

func (uc *DeleteForwardRuleUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.ruleRepo.Delete(ctx, id)
	})
	if err != nil {
		return fmt.Errorf("deleteForwardRule: %w", err)
	}
	return nil
}

// ListForwardDeliveriesUseCase is the admin-facing delivery log -- "did my
// fan-out rule actually fire, and did it succeed" -- read by the
// Integrations page.
type ListForwardDeliveriesUseCase struct {
	db           *gorm.DB
	deliveryRepo webhooks.ForwardDeliveryRepository
}

func NewListForwardDeliveriesUseCase(db *gorm.DB, deliveryRepo webhooks.ForwardDeliveryRepository) *ListForwardDeliveriesUseCase {
	return &ListForwardDeliveriesUseCase{db: db, deliveryRepo: deliveryRepo}
}

func (uc *ListForwardDeliveriesUseCase) Execute(ctx context.Context, limit int) ([]webhooks.ForwardDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var out []webhooks.ForwardDelivery
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.deliveryRepo.ListRecent(ctx, limit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listForwardDeliveries: %w", err)
	}
	return out, nil
}
