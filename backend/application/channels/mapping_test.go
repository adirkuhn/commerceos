package channels_test

import (
	"testing"
	"time"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestMapWooStatus(t *testing.T) {
	cases := map[string]pmsDomain.ProductStatus{
		"publish": pmsDomain.ProductStatusActive,
		"draft":   pmsDomain.ProductStatusDraft,
		"pending": pmsDomain.ProductStatusDraft,
		"private": pmsDomain.ProductStatusArchived,
		"trash":   pmsDomain.ProductStatusArchived,
		"":        pmsDomain.ProductStatusDraft,
		"unknown": pmsDomain.ProductStatusDraft,
	}
	for wooStatus, want := range cases {
		if got := appchannels.MapWooStatus(wooStatus); got != want {
			t.Errorf("MapWooStatus(%q) = %q, want %q", wooStatus, got, want)
		}
	}
}

func TestShouldImportWooProductType(t *testing.T) {
	if !appchannels.ShouldImportWooProductType("simple") {
		t.Error("simple products must be importable")
	}
	for _, skip := range []string{"variable", "grouped", "external", ""} {
		if appchannels.ShouldImportWooProductType(skip) {
			t.Errorf("type %q must be skipped this phase (only simple products are mapped)", skip)
		}
	}
}

func TestParseWooPriceToMinorUnits(t *testing.T) {
	cases := map[string]int64{
		"1.49":  149,
		"10":    1000,
		"0":     0,
		"":      0,
		"19.99": 1999,
		"1.005": 101, // rounds to nearest cent
	}
	for price, want := range cases {
		got, err := appchannels.ParseWooPriceToMinorUnits(price)
		if err != nil {
			t.Errorf("ParseWooPriceToMinorUnits(%q) unexpected error: %v", price, err)
			continue
		}
		if got != want {
			t.Errorf("ParseWooPriceToMinorUnits(%q) = %d, want %d", price, got, want)
		}
	}
}

func TestParseWooPriceToMinorUnits_rejectsGarbage(t *testing.T) {
	if _, err := appchannels.ParseWooPriceToMinorUnits("not-a-price"); err == nil {
		t.Error("expected error for unparseable price")
	}
}

func TestMapCommerceStatusToWoo(t *testing.T) {
	cases := map[pmsDomain.ProductStatus]string{
		pmsDomain.ProductStatusActive:   "publish",
		pmsDomain.ProductStatusDraft:    "draft",
		pmsDomain.ProductStatusArchived: "private",
	}
	for status, want := range cases {
		if got := appchannels.MapCommerceStatusToWoo(status); got != want {
			t.Errorf("MapCommerceStatusToWoo(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestFormatMinorUnitsAsWooPrice(t *testing.T) {
	cases := map[int64]string{
		149:  "1.49",
		1000: "10.00",
		0:    "0.00",
		1999: "19.99",
		5:    "0.05",
	}
	for minorUnits, want := range cases {
		if got := appchannels.FormatMinorUnitsAsWooPrice(minorUnits); got != want {
			t.Errorf("FormatMinorUnitsAsWooPrice(%d) = %q, want %q", minorUnits, got, want)
		}
	}
}

func TestFormatMinorUnitsAsWooPrice_roundTripsWithParse(t *testing.T) {
	for _, minorUnits := range []int64{149, 1000, 0, 1999, 5, 123456} {
		formatted := appchannels.FormatMinorUnitsAsWooPrice(minorUnits)
		parsed, err := appchannels.ParseWooPriceToMinorUnits(formatted)
		if err != nil {
			t.Fatalf("ParseWooPriceToMinorUnits(%q) unexpected error: %v", formatted, err)
		}
		if parsed != minorUnits {
			t.Errorf("round-trip %d -> %q -> %d, want %d", minorUnits, formatted, parsed, minorUnits)
		}
	}
}

func TestWooSaleFields_noSaleReturnsBothEmpty(t *testing.T) {
	salePrice, saleEndDate := appchannels.WooSaleFields(pmsDomain.SKU{PriceMinorUnits: 1000})
	if salePrice != "" || saleEndDate != "" {
		t.Errorf("got (%q, %q), want (\"\", \"\")", salePrice, saleEndDate)
	}
}

func TestWooSaleFields_salePriceWithoutEndDate(t *testing.T) {
	sale := int64(800)
	salePrice, saleEndDate := appchannels.WooSaleFields(pmsDomain.SKU{PriceMinorUnits: 1000, SalePriceMinorUnits: &sale})
	if salePrice != "8.00" {
		t.Errorf("salePrice = %q, want \"8.00\"", salePrice)
	}
	if saleEndDate != "" {
		t.Errorf("saleEndDate = %q, want \"\"", saleEndDate)
	}
}

func TestWooSaleFields_salePriceWithEndDate(t *testing.T) {
	sale := int64(800)
	end := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	salePrice, saleEndDate := appchannels.WooSaleFields(pmsDomain.SKU{PriceMinorUnits: 1000, SalePriceMinorUnits: &sale, SaleEndDate: &end})
	if salePrice != "8.00" {
		t.Errorf("salePrice = %q, want \"8.00\"", salePrice)
	}
	if saleEndDate != "2026-08-10" {
		t.Errorf("saleEndDate = %q, want \"2026-08-10\"", saleEndDate)
	}
}
