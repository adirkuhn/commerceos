package collections

import (
	"github.com/google/uuid"

	gencollections "github.com/adirkuhn/commerceos/api/collections"
	"github.com/adirkuhn/commerceos/domain/collections"
)

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

func fromInput(r gencollections.CollectionInput) collections.Draft {
	d := collections.Draft{
		Name: r.Name, Slug: r.Slug, Description: deref(r.Description), SEOTitle: deref(r.SeoTitle),
		SEODescription: deref(r.SeoDescription), Type: collections.Type(r.Type), Sort: collections.Sort(r.Sort),
		Published: deref(r.Published),
	}
	if r.ImageAssetId != nil {
		id := r.ImageAssetId.String()
		d.ImageAssetID = &id
	}
	if r.Match != nil {
		d.Match = collections.Match(*r.Match)
	}
	for _, rule := range deref(r.Rules) {
		d.Rules = append(d.Rules, collections.Rule{
			Field: collections.RuleField(rule.Field), Value: deref(rule.Value), IncludeSubcategories: deref(rule.IncludeSubcategories),
			MinMinorUnits: rule.MinMinorUnits, MaxMinorUnits: rule.MaxMinorUnits, Days: deref(rule.Days),
		})
	}
	return d
}

func toGen(c collections.Collection, productIDs []string) gencollections.Collection {
	rules := make([]gencollections.CollectionRule, len(c.Rules))
	for i, r := range c.Rules {
		rule := gencollections.CollectionRule{Field: gencollections.CollectionRuleField(r.Field), MinMinorUnits: r.MinMinorUnits, MaxMinorUnits: r.MaxMinorUnits}
		if r.Value != "" {
			v := r.Value
			rule.Value = &v
		}
		if r.IncludeSubcategories {
			b := true
			rule.IncludeSubcategories = &b
		}
		if r.Days != 0 {
			n := r.Days
			rule.Days = &n
		}
		rules[i] = rule
	}
	id, _ := uuid.Parse(c.ID) // ids come from the database, always UUIDs
	out := gencollections.Collection{
		Id: id, Name: c.Name, Slug: c.Slug, Description: c.Description, SeoTitle: c.SEOTitle,
		SeoDescription: c.SEODescription, Type: gencollections.CollectionType(c.Type), Match: gencollections.CollectionMatch(c.Match),
		Rules: rules, Sort: gencollections.CollectionSort(c.Sort), Published: c.Published, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	if c.ImageAssetID != nil {
		id, err := uuid.Parse(*c.ImageAssetID)
		if err == nil {
			out.ImageAssetId = &id
		}
	}
	if productIDs != nil {
		ids := make([]uuid.UUID, 0, len(productIDs))
		for _, p := range productIDs {
			if id, err := uuid.Parse(p); err == nil {
				ids = append(ids, id)
			}
		}
		out.ProductIds = &ids
	}
	return out
}
