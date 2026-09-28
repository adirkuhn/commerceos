// Package collections is merchandising groups of products ("Summer sale",
// "New arrivals") shown on the storefront, distinct from categories (the
// product taxonomy). A collection is manual (hand-picked, ordered) or
// smart (products matching rules, evaluated when read -- no membership to
// keep in sync). Collections belong to the tenant; each storefront shows
// only the products enrolled on its own channel.
package collections

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidCollection  = errors.New("invalid collection")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrSlugTaken          = errors.New("another collection already uses this URL")
	ErrNotManual          = errors.New("only a manual collection has hand-picked products")
)

type Type string

const (
	TypeManual Type = "manual"
	TypeSmart  Type = "smart"
)

type Match string

const (
	MatchAll Match = "all"
	MatchAny Match = "any"
)

type Sort string

const (
	SortManual    Sort = "manual"
	SortNameAsc   Sort = "name_asc"
	SortPriceAsc  Sort = "price_asc"
	SortPriceDesc Sort = "price_desc"
	SortNewest    Sort = "newest"
)

type RuleField string

const (
	FieldCategory        RuleField = "category"
	FieldTag             RuleField = "tag"
	FieldPrice           RuleField = "price"
	FieldOnSale          RuleField = "on_sale"
	FieldAddedWithinDays RuleField = "added_within_days"
)

const (
	MaxRules          = 10
	MaxManualProducts = 1000
	maxNameLength     = 120
	maxSlugLength     = 64
	maxSEOTitle       = 70
	maxSEODescription = 320
	maxDescription    = 5000
	maxDays           = 3650
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Rule is one smart-collection condition. Which fields apply depends on
// Field: Value (category id / tag), IncludeSubcategories (category),
// MinMinorUnits/MaxMinorUnits (price, either may be open), Days
// (added_within_days). on_sale takes nothing.
type Rule struct {
	Field                RuleField
	Value                string
	IncludeSubcategories bool
	MinMinorUnits        *int64
	MaxMinorUnits        *int64
	Days                 int
}

// Draft is everything a merchant sets on a collection.
type Draft struct {
	Name           string
	Slug           string
	Description    string
	ImageAssetID   *string
	SEOTitle       string
	SEODescription string
	Type           Type
	Match          Match
	Rules          []Rule
	Sort           Sort
	Published      bool
}

type Collection struct {
	ID       string
	TenantID string
	Draft
	CreatedAt time.Time
	UpdatedAt time.Time
}

func New(tenantID string, d Draft, now time.Time) (Collection, error) {
	d, err := validate(d)
	if err != nil {
		return Collection{}, err
	}
	return Collection{TenantID: tenantID, Draft: d, CreatedAt: now, UpdatedAt: now}, nil
}

func (c *Collection) Update(d Draft, now time.Time) error {
	d, err := validate(d)
	if err != nil {
		return err
	}
	c.Draft = d
	c.UpdatedAt = now
	return nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidCollection, fmt.Sprintf(format, args...))
}

func validate(d Draft) (Draft, error) {
	d.Name = strings.TrimSpace(d.Name)
	if d.Name == "" || utf8.RuneCountInString(d.Name) > maxNameLength {
		return d, invalid("name must be 1-%d characters", maxNameLength)
	}
	if len(d.Slug) > maxSlugLength || !slugPattern.MatchString(d.Slug) {
		return d, invalid("URL must be lowercase letters, digits and dashes")
	}
	if utf8.RuneCountInString(d.Description) > maxDescription {
		return d, invalid("description is too long")
	}
	if utf8.RuneCountInString(d.SEOTitle) > maxSEOTitle || utf8.RuneCountInString(d.SEODescription) > maxSEODescription {
		return d, invalid("search engine title/description is too long")
	}
	switch d.Sort {
	case SortManual, SortNameAsc, SortPriceAsc, SortPriceDesc, SortNewest:
	default:
		return d, invalid("unknown sort %q", d.Sort)
	}
	switch d.Type {
	case TypeManual:
		if len(d.Rules) > 0 {
			return d, invalid("a manual collection has no rules")
		}
	case TypeSmart:
		if d.Sort == SortManual {
			return d, invalid("a smart collection can't be sorted by hand")
		}
		if d.Match != MatchAll && d.Match != MatchAny {
			return d, invalid("match must be all or any")
		}
		if len(d.Rules) == 0 || len(d.Rules) > MaxRules {
			return d, invalid("a smart collection needs 1-%d rules", MaxRules)
		}
		for i, r := range d.Rules {
			if err := validateRule(r); err != nil {
				return d, fmt.Errorf("rule %d: %w", i+1, err)
			}
		}
	default:
		return d, invalid("unknown type %q", d.Type)
	}
	return d, nil
}

func validateRule(r Rule) error {
	switch r.Field {
	case FieldCategory, FieldTag:
		if strings.TrimSpace(r.Value) == "" {
			return invalid("%s needs a value", r.Field)
		}
	case FieldPrice:
		if r.MinMinorUnits == nil && r.MaxMinorUnits == nil {
			return invalid("price needs a minimum or maximum")
		}
		if (r.MinMinorUnits != nil && *r.MinMinorUnits < 0) || (r.MaxMinorUnits != nil && *r.MaxMinorUnits < 0) ||
			(r.MinMinorUnits != nil && r.MaxMinorUnits != nil && *r.MinMinorUnits > *r.MaxMinorUnits) {
			return invalid("price range is invalid")
		}
	case FieldOnSale:
	case FieldAddedWithinDays:
		if r.Days < 1 || r.Days > maxDays {
			return invalid("days must be 1-%d", maxDays)
		}
	default:
		return invalid("unknown rule %q", r.Field)
	}
	return nil
}

// ValidateManualProducts checks a manual collection's ordered product list.
func ValidateManualProducts(ids []string) error {
	if len(ids) > MaxManualProducts {
		return invalid("at most %d products", MaxManualProducts)
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return invalid("product %s is listed twice", id)
		}
		seen[id] = true
	}
	return nil
}
