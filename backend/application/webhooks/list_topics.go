package webhooks

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// providerWebhookSubscriptionsLister is application/integrations'
// ListAllProviderWebhooksUseCase, narrowed to point of use (ISP) --
// injected rather than re-deriving Woo/MyParcel client-building logic here.
// Cross-domain use-case-to-use-case call, matching CLAUDE.md's "cross-domain
// calls go through use cases" rule.
type providerWebhookSubscriptionsLister interface {
	Execute(ctx context.Context) (*appintegrations.ListAllProviderWebhooksResult, error)
}

// TopicsByProvider maps provider -> its known topics, sorted, deduplicated,
// never including "" (the fan-out form's own "All events" wildcard option
// is added client-side, not part of this list).
type TopicsByProvider map[string][]string

// ListAvailableWebhookTopicsUseCase answers "what topics could I actually
// write a fan-out rule for" -- the union of three sources: the baseline
// catalog (integrationsDomain.KnownWebhookTopicsByProvider, so a brand-new
// tenant with zero traffic still gets a populated dropdown instead of an
// empty one), every topic this tenant has ever actually received a webhook
// event for (WebhookEventRepository, real traffic), and every topic
// currently subscribed at the provider (providerWebhookSubscriptionsLister,
// what's about to start arriving).
type ListAvailableWebhookTopicsUseCase struct {
	db                *gorm.DB
	eventRepo         webhooks.WebhookEventRepository
	subscriptionsList providerWebhookSubscriptionsLister
}

func NewListAvailableWebhookTopicsUseCase(
	db *gorm.DB, eventRepo webhooks.WebhookEventRepository, subscriptionsList providerWebhookSubscriptionsLister,
) *ListAvailableWebhookTopicsUseCase {
	return &ListAvailableWebhookTopicsUseCase{db: db, eventRepo: eventRepo, subscriptionsList: subscriptionsList}
}

func (uc *ListAvailableWebhookTopicsUseCase) Execute(ctx context.Context) (TopicsByProvider, error) {
	sets := map[string]map[string]struct{}{}
	add := func(provider, topic string) {
		if provider == "" || topic == "" {
			return
		}
		if sets[provider] == nil {
			sets[provider] = map[string]struct{}{}
		}
		sets[provider][topic] = struct{}{}
	}

	for provider, topics := range integrationsDomain.KnownWebhookTopicsByProvider() {
		for _, topic := range topics {
			add(string(provider), topic)
		}
	}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		refs, err := uc.eventRepo.DistinctTopics(ctx)
		if err != nil {
			return err
		}
		for _, ref := range refs {
			add(ref.Provider, ref.Topic)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listAvailableWebhookTopics: %w", err)
	}

	// Subscriptions is a best-effort second source -- a live provider API
	// failure here (already surfaced to the admin on the Subscriptions tab)
	// must not blank out the topics already known from real received
	// traffic.
	if result, err := uc.subscriptionsList.Execute(ctx); err == nil {
		for _, w := range result.Webhooks {
			add(string(w.Provider), w.Topic)
		}
	}

	out := make(TopicsByProvider, len(sets))
	for provider, set := range sets {
		topics := make([]string, 0, len(set))
		for topic := range set {
			topics = append(topics, topic)
		}
		sort.Strings(topics)
		out[provider] = topics
	}
	return out, nil
}
