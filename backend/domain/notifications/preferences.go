package notifications

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
)

var (
	ErrInvalidRecipient        = errors.New("notification email recipient is not a valid address")
	ErrTooManyRecipients       = errors.New("at most 10 notification email recipients")
	ErrEmailRecipientsRequired = errors.New("add at least one email recipient to send notifications by email")
	ErrUnknownCategory         = errors.New("unknown notification category")
)

// Category groups notification Types into the buckets people choose
// delivery for ("tell me about imports by email"). Types stay open-ended;
// CategoryOf maps them by prefix.
type Category string

const (
	CategoryImports           Category = "imports"
	CategoryPurchaseDocuments Category = "purchase_documents"
	CategoryChannelOffers     Category = "channel_offers"
	CategoryOther             Category = "other"
)

// Categories lists every category, in display order.
var Categories = []Category{CategoryImports, CategoryPurchaseDocuments, CategoryChannelOffers, CategoryOther}

const maxRecipients = 10

var categoryPrefixes = []struct {
	prefix   string
	category Category
}{
	{"import.", CategoryImports},
	{"procurement.", CategoryPurchaseDocuments},
	{"bol.", CategoryChannelOffers},
}

func CategoryOf(t Type) Category {
	for _, p := range categoryPrefixes {
		if strings.HasPrefix(string(t), p.prefix) {
			return p.category
		}
	}
	return CategoryOther
}

// ChannelPrefs says where one category's notifications go.
type ChannelPrefs struct {
	InApp bool // the bell in the admin
	Email bool // to Preferences.EmailRecipients
}

var defaultChannelPrefs = ChannelPrefs{InApp: true, Email: false}

// Preferences is a tenant's notification delivery choice per category.
type Preferences struct {
	TenantID        string
	Categories      map[Category]ChannelPrefs
	EmailRecipients []string
	UpdatedAt       time.Time
}

type PreferencesInput struct {
	Categories      map[Category]ChannelPrefs
	EmailRecipients []string
}

type PreferencesRepository interface {
	// Get returns the saved preferences, or nil when none were saved.
	Get(ctx context.Context) (*Preferences, error)
	Upsert(ctx context.Context, p *Preferences) error
}

// DefaultPreferences is what a tenant gets before choosing: every category
// in the bell, nothing by email -- today's behaviour.
func DefaultPreferences(tenantID string) *Preferences {
	cats := make(map[Category]ChannelPrefs, len(Categories))
	for _, c := range Categories {
		cats[c] = defaultChannelPrefs
	}
	return &Preferences{TenantID: tenantID, Categories: cats, EmailRecipients: []string{}}
}

func NewPreferences(tenantID string, in PreferencesInput, now time.Time) (*Preferences, error) {
	p := DefaultPreferences(tenantID)
	p.UpdatedAt = now
	for c, prefs := range in.Categories {
		if !isKnownCategory(c) {
			return nil, ErrUnknownCategory
		}
		p.Categories[c] = prefs
	}

	seen := map[string]bool{}
	for _, raw := range in.EmailRecipients {
		addr := strings.TrimSpace(raw)
		if addr == "" || seen[strings.ToLower(addr)] {
			continue
		}
		if parsed, err := mail.ParseAddress(addr); err != nil || parsed.Address != addr {
			return nil, ErrInvalidRecipient
		}
		seen[strings.ToLower(addr)] = true
		p.EmailRecipients = append(p.EmailRecipients, addr)
	}
	if len(p.EmailRecipients) > maxRecipients {
		return nil, ErrTooManyRecipients
	}
	if len(p.EmailRecipients) == 0 {
		for _, prefs := range p.Categories {
			if prefs.Email {
				return nil, ErrEmailRecipientsRequired
			}
		}
	}
	return p, nil
}

func (p *Preferences) For(c Category) ChannelPrefs {
	if prefs, ok := p.Categories[c]; ok {
		return prefs
	}
	return defaultChannelPrefs
}

func (p *Preferences) ForType(t Type) ChannelPrefs { return p.For(CategoryOf(t)) }

func isKnownCategory(c Category) bool {
	for _, k := range Categories {
		if k == c {
			return true
		}
	}
	return false
}
