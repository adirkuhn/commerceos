package collections_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/collections"
)

var now = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func valid() collections.Draft {
	return collections.Draft{Name: " Summer sale ", Slug: "summer-sale", Type: collections.TypeManual, Sort: collections.SortManual}
}

func TestNew_manual(t *testing.T) {
	c, err := collections.New("tenant-1", valid(), now)
	require.NoError(t, err)
	assert.Equal(t, "Summer sale", c.Name)
	assert.False(t, c.Published)
}

func TestNew_validation(t *testing.T) {
	cases := map[string]func(*collections.Draft){
		"empty name":        func(d *collections.Draft) { d.Name = " " },
		"long name":         func(d *collections.Draft) { d.Name = strings.Repeat("x", 121) },
		"bad slug":          func(d *collections.Draft) { d.Slug = "Summer Sale" },
		"unknown type":      func(d *collections.Draft) { d.Type = "auto" },
		"unknown sort":      func(d *collections.Draft) { d.Sort = "random" },
		"manual with rules": func(d *collections.Draft) { d.Rules = []collections.Rule{{Field: collections.FieldOnSale}} },
		"smart without rules": func(d *collections.Draft) {
			d.Type = collections.TypeSmart
			d.Sort = collections.SortNewest
			d.Match = collections.MatchAll
		},
		"smart manual sort": func(d *collections.Draft) {
			d.Type = collections.TypeSmart
			d.Match = collections.MatchAll
			d.Rules = []collections.Rule{{Field: collections.FieldOnSale}}
		},
		"smart bad match": func(d *collections.Draft) {
			d.Type = collections.TypeSmart
			d.Sort = collections.SortNewest
			d.Rules = []collections.Rule{{Field: collections.FieldOnSale}}
		},
		"long seo description": func(d *collections.Draft) { d.SEODescription = strings.Repeat("x", 321) },
	}
	for name, mutate := range cases {
		d := valid()
		mutate(&d)
		_, err := collections.New("tenant-1", d, now)
		assert.ErrorIs(t, err, collections.ErrInvalidCollection, name)
	}
}

func TestRules_validation(t *testing.T) {
	min, max := int64(1000), int64(500)
	bad := []collections.Rule{
		{Field: collections.FieldCategory},
		{Field: collections.FieldTag, Value: " "},
		{Field: collections.FieldPrice},
		{Field: collections.FieldPrice, MinMinorUnits: &min, MaxMinorUnits: &max},
		{Field: collections.FieldAddedWithinDays, Days: 0},
		{Field: collections.FieldAddedWithinDays, Days: 4000},
		{Field: "stock"},
	}
	for _, r := range bad {
		d := valid()
		d.Type, d.Sort, d.Match, d.Rules = collections.TypeSmart, collections.SortNewest, collections.MatchAll, []collections.Rule{r}
		_, err := collections.New("tenant-1", d, now)
		assert.ErrorIs(t, err, collections.ErrInvalidCollection, "%+v", r)
	}

	good := []collections.Rule{
		{Field: collections.FieldCategory, Value: "cat-1", IncludeSubcategories: true},
		{Field: collections.FieldTag, Value: "summer"},
		{Field: collections.FieldPrice, MinMinorUnits: &max},
		{Field: collections.FieldOnSale},
		{Field: collections.FieldAddedWithinDays, Days: 30},
	}
	d := valid()
	d.Type, d.Sort, d.Match, d.Rules = collections.TypeSmart, collections.SortPriceAsc, collections.MatchAny, good
	_, err := collections.New("tenant-1", d, now)
	assert.NoError(t, err)

	d.Rules = make([]collections.Rule, collections.MaxRules+1)
	for i := range d.Rules {
		d.Rules[i] = collections.Rule{Field: collections.FieldOnSale}
	}
	_, err = collections.New("tenant-1", d, now)
	assert.ErrorIs(t, err, collections.ErrInvalidCollection)
}

func TestUpdate_keepsIdentity(t *testing.T) {
	c, _ := collections.New("tenant-1", valid(), now)
	c.ID = "c1"
	d := valid()
	d.Name = "Winter"
	d.Published = true
	later := now.Add(time.Hour)

	require.NoError(t, c.Update(d, later))

	assert.Equal(t, "c1", c.ID)
	assert.Equal(t, "Winter", c.Name)
	assert.True(t, c.Published)
	assert.Equal(t, later, c.UpdatedAt)
	assert.Equal(t, now, c.CreatedAt)

	d.Slug = "BAD"
	assert.ErrorIs(t, c.Update(d, later), collections.ErrInvalidCollection)
	assert.Equal(t, "Winter", c.Name, "failed update changes nothing")
}

func TestValidateManualProducts(t *testing.T) {
	assert.NoError(t, collections.ValidateManualProducts([]string{"a", "b"}))
	assert.ErrorIs(t, collections.ValidateManualProducts([]string{"a", "a"}), collections.ErrInvalidCollection)
	ids := make([]string, collections.MaxManualProducts+1)
	for i := range ids {
		ids[i] = strings.Repeat("x", i%7+1) + string(rune('a'+i%26)) + strings.Repeat("y", i/26)
	}
	assert.ErrorIs(t, collections.ValidateManualProducts(ids), collections.ErrInvalidCollection)
}
