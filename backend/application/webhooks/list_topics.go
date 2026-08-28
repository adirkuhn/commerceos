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

// TopicsByProvider maps provider -> its topics, sorted, deduplicated, never
// including "" (the fan-out form's own "All events" wildcard option is
// added client-side, not part of this list).
type TopicsByProvider map[string][]string

// AvailableWebhookTopics is ListAvailableWebhookTopicsUseCase's result,
// split into the two groups the fan-out rule form's Topic dropdown offers
// as separate option groups:
//   - AllEvents: every kind of event that could ever happen for a
//     provider -- the baseline catalog (integrationsDomain.
//     KnownWebhookTopicsByProvider) unioned with every topic this tenant
//     has actually received (WebhookEventRepository). A reference list;
//     picking one here doesn't guarantee it's actually being delivered.
//   - ActiveWebhooks: topics with a currently active subscription
//     registered at the provider right now (providerWebhookSubscriptionsLister,
//     status == "active") -- what's actually about to arrive, i.e. what a
//     fan-out rule written against it will really fire for.
type AvailableWebhookTopics struct {
	AllEvents      TopicsByProvider
	ActiveWebhooks TopicsByProvider
}

// ListAvailableWebhookTopicsUseCase answers "what topics could I actually
// write a fan-out rule for". See AvailableWebhookTopics' doc comment for
// how the two groups it returns differ.
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

func (uc *ListAvailableWebhookTopicsUseCase) Execute(ctx context.Context) (*AvailableWebhookTopics, error) {
	allEvents := newTopicSet()
	for provider, topics := range integrationsDomain.KnownWebhookTopicsByProvider() {
		for _, topic := range topics {
			allEvents.add(string(provider), topic)
		}
	}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		refs, err := uc.eventRepo.DistinctTopics(ctx)
		if err != nil {
			return err
		}
		for _, ref := range refs {
			allEvents.add(ref.Provider, ref.Topic)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listAvailableWebhookTopics: %w", err)
	}

	activeWebhooks := newTopicSet()
	// Subscriptions is a best-effort source -- a live provider API failure
	// here (already surfaced to the admin on the Subscriptions tab) just
	// means an empty ActiveWebhooks group, not a failed request.
	if result, err := uc.subscriptionsList.Execute(ctx); err == nil {
		for _, w := range result.Webhooks {
			if w.Status != "active" {
				continue
			}
			activeWebhooks.add(string(w.Provider), w.Topic)
		}
	}

	return &AvailableWebhookTopics{AllEvents: allEvents.result(), ActiveWebhooks: activeWebhooks.result()}, nil
}

// topicSet accumulates provider -> topic pairs, deduplicated, until
// result() sorts and finalizes them into a TopicsByProvider.
type topicSet map[string]map[string]struct{}

func newTopicSet() topicSet { return topicSet{} }

func (s topicSet) add(provider, topic string) {
	if provider == "" || topic == "" {
		return
	}
	if s[provider] == nil {
		s[provider] = map[string]struct{}{}
	}
	s[provider][topic] = struct{}{}
}

func (s topicSet) result() TopicsByProvider {
	out := make(TopicsByProvider, len(s))
	for provider, set := range s {
		topics := make([]string, 0, len(set))
		for topic := range set {
			topics = append(topics, topic)
		}
		sort.Strings(topics)
		out[provider] = topics
	}
	return out
}
