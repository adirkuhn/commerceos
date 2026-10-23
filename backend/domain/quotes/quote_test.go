package quotes_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/quotes"
)

var now = time.Date(2026, 10, 23, 10, 0, 0, 0, time.UTC)

func line(sku string, qty, price int64) quotes.Line {
	return quotes.Line{ProductID: "p-" + sku, SKUID: sku, Quantity: qty, UnitPriceMinorUnits: price}
}

func validInput() quotes.Input {
	return quotes.Input{Name: "  Restaurant Sabor  ", DiscountBasisPoints: 3000, CurrencyCode: "EUR",
		Lines: []quotes.Line{line("s-1", 100, 871), line("s-2", 5, 1250)}}
}

func TestNewQuote(t *testing.T) {
	q, err := quotes.NewQuote("t-1", validInput(), "user-1", now)

	require.NoError(t, err)
	assert.Equal(t, "Restaurant Sabor", q.Name, "name is trimmed")
	assert.Equal(t, "t-1", q.TenantID)
	assert.Equal(t, "user-1", q.CreatedBy)
	assert.Equal(t, now, q.CreatedAt)
	assert.Equal(t, now, q.UpdatedAt)
	assert.Equal(t, int64(100*871+5*1250), q.Total())
}

func TestNewQuote_validation(t *testing.T) {
	cases := map[string]struct {
		edit func(*quotes.Input)
		want error
	}{
		"blank name":         {func(in *quotes.Input) { in.Name = "  " }, quotes.ErrNameRequired},
		"long name":          {func(in *quotes.Input) { in.Name = strings.Repeat("x", 201) }, quotes.ErrNameTooLong},
		"negative discount":  {func(in *quotes.Input) { in.DiscountBasisPoints = -1 }, quotes.ErrInvalidDiscount},
		"discount over 100%": {func(in *quotes.Input) { in.DiscountBasisPoints = 10001 }, quotes.ErrInvalidDiscount},
		"bad currency":       {func(in *quotes.Input) { in.CurrencyCode = "eu" }, quotes.ErrInvalidCurrency},
		"negative quantity":  {func(in *quotes.Input) { in.Lines[0].Quantity = -1 }, quotes.ErrInvalidLine},
		"negative price":     {func(in *quotes.Input) { in.Lines[0].UnitPriceMinorUnits = -1 }, quotes.ErrInvalidLine},
		"line without sku":   {func(in *quotes.Input) { in.Lines[0].SKUID = "" }, quotes.ErrInvalidLine},
		"same sku twice":     {func(in *quotes.Input) { in.Lines[1].SKUID = in.Lines[0].SKUID }, quotes.ErrDuplicateLine},
		"too many lines": {func(in *quotes.Input) {
			in.Lines = make([]quotes.Line, quotes.MaxLines+1)
			for i := range in.Lines {
				in.Lines[i] = line(strings.Repeat("s", i+1), 1, 1)
			}
		}, quotes.ErrTooManyLines},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			in := validInput()
			c.edit(&in)
			_, err := quotes.NewQuote("t-1", in, "user-1", now)
			assert.ErrorIs(t, err, c.want)
		})
	}
}

func TestNewQuote_emptyQuoteWithoutCurrencyIsFine(t *testing.T) {
	_, err := quotes.NewQuote("t-1", quotes.Input{Name: "Draft"}, "", now)
	assert.NoError(t, err)
}

func TestQuoteRevise_replacesContentKeepsIdentity(t *testing.T) {
	q, err := quotes.NewQuote("t-1", validInput(), "user-1", now)
	require.NoError(t, err)
	q.ID, q.Number = "q-1", 7
	later := now.Add(time.Hour)

	err = q.Revise(quotes.Input{Name: "Renamed", DiscountBasisPoints: 0, CurrencyCode: "EUR", Lines: []quotes.Line{line("s-3", 2, 500)}}, later)

	require.NoError(t, err)
	assert.Equal(t, "q-1", q.ID)
	assert.Equal(t, int64(7), q.Number)
	assert.Equal(t, now, q.CreatedAt)
	assert.Equal(t, later, q.UpdatedAt)
	assert.Equal(t, "Renamed", q.Name)
	assert.Equal(t, int64(1000), q.Total())
}

func TestQuoteRevise_invalidLeavesQuoteUntouched(t *testing.T) {
	q, err := quotes.NewQuote("t-1", validInput(), "user-1", now)
	require.NoError(t, err)

	err = q.Revise(quotes.Input{Name: ""}, now.Add(time.Hour))

	assert.ErrorIs(t, err, quotes.ErrNameRequired)
	assert.Equal(t, "Restaurant Sabor", q.Name)
	assert.Len(t, q.Lines, 2)
}

func TestReference(t *testing.T) {
	assert.Equal(t, "Q-0007", quotes.Reference(7))
	assert.Equal(t, "Q-12345", quotes.Reference(12345))
}
