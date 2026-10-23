// Package quotes is a saved B2B price quote: named lines of SKUs with the
// price agreed for each, kept as-is once saved -- reopening a quote shows
// what was offered, not today's price.
package quotes

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxLines      = 500
	maxNameLength = 200
	fullDiscount  = 10000
)

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// Line is one SKU on a quote. UnitPriceMinorUnits is the price offered,
// stored -- not recomputed from the catalog.
type Line struct {
	ProductID           string
	SKUID               string
	Quantity            int64
	UnitPriceMinorUnits int64
}

type Quote struct {
	ID       string
	TenantID string
	// Number is the shop's running quote number (see Reference), assigned
	// when the quote is first stored; 0 before that.
	Number int64
	Name   string
	// DiscountBasisPoints is the discount the quote was built with (3000 =
	// 30%), so reopening it shows the same starting point.
	DiscountBasisPoints int
	// CurrencyCode is "" only for a quote without lines.
	CurrencyCode string
	Lines        []Line
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Input is a quote's editable content -- what both creating and saving
// over an existing quote take.
type Input struct {
	Name                string
	DiscountBasisPoints int
	CurrencyCode        string
	Lines               []Line
}

func NewQuote(tenantID string, in Input, createdBy string, now time.Time) (*Quote, error) {
	q := &Quote{TenantID: tenantID, CreatedBy: createdBy, CreatedAt: now}
	if err := q.Revise(in, now); err != nil {
		return nil, err
	}
	return q, nil
}

// Revise replaces the quote's content; on an invalid input the quote is
// left as it was.
func (q *Quote) Revise(in Input, now time.Time) error {
	name := strings.TrimSpace(in.Name)
	if err := validate(name, in); err != nil {
		return err
	}
	q.Name = name
	q.DiscountBasisPoints = in.DiscountBasisPoints
	q.CurrencyCode = in.CurrencyCode
	q.Lines = append([]Line(nil), in.Lines...)
	q.UpdatedAt = now
	return nil
}

func validate(name string, in Input) error {
	switch {
	case name == "":
		return ErrNameRequired
	case utf8.RuneCountInString(name) > maxNameLength:
		return ErrNameTooLong
	case in.DiscountBasisPoints < 0 || in.DiscountBasisPoints > fullDiscount:
		return ErrInvalidDiscount
	case (in.CurrencyCode != "" || len(in.Lines) > 0) && !currencyCode.MatchString(in.CurrencyCode):
		return ErrInvalidCurrency
	case len(in.Lines) > MaxLines:
		return ErrTooManyLines
	}
	seen := make(map[string]bool, len(in.Lines))
	for _, l := range in.Lines {
		if l.SKUID == "" || l.ProductID == "" || l.Quantity < 0 || l.UnitPriceMinorUnits < 0 {
			return ErrInvalidLine
		}
		if seen[l.SKUID] {
			return ErrDuplicateLine
		}
		seen[l.SKUID] = true
	}
	return nil
}

// Total is the sum of every line's quantity times its price.
func (q Quote) Total() int64 {
	var total int64
	for _, l := range q.Lines {
		total += l.Quantity * l.UnitPriceMinorUnits
	}
	return total
}

// Reference is how a quote number is shown: Q-0007.
func Reference(number int64) string {
	return fmt.Sprintf("Q-%04d", number)
}
