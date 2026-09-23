package settings_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/settings"
)

var now = time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)

func validInput() settings.CompanyProfileInput {
	return settings.CompanyProfileInput{
		LegalName:         "Cantinho Brasileiro B.V.",
		TradingName:       "Cantinho",
		VATNumber:         "NL123456789B01",
		ChamberOfCommerce: "12345678",
		Email:             "info@example.nl",
		Phone:             "+31 20 123 4567",
		Address: settings.CompanyAddress{
			Street: "Damrak", HouseNumber: "1", PostalCode: "1012 LG", City: "Amsterdam", Country: "NL",
		},
		IBAN: "NL91ABNA0417164300",
	}
}

func TestNewCompanyProfile_ValidInput(t *testing.T) {
	p, err := settings.NewCompanyProfile("tenant-1", validInput(), now)

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", p.TenantID)
	assert.Equal(t, "Cantinho Brasileiro B.V.", p.LegalName)
	assert.Equal(t, "NL", p.Address.Country)
	assert.Equal(t, now, p.UpdatedAt)
}

func TestNewCompanyProfile_Normalizes(t *testing.T) {
	in := validInput()
	in.LegalName = "  Cantinho B.V.  "
	in.VATNumber = "nl 1234.5678.9b01"
	in.IBAN = "nl91 abna 0417 1643 00"
	in.Address.Country = "nl"
	in.Email = "  Info@Example.nl "

	p, err := settings.NewCompanyProfile("tenant-1", in, now)

	require.NoError(t, err)
	assert.Equal(t, "Cantinho B.V.", p.LegalName)
	assert.Equal(t, "NL123456789B01", p.VATNumber)
	assert.Equal(t, "NL91ABNA0417164300", p.IBAN)
	assert.Equal(t, "NL", p.Address.Country)
	assert.Equal(t, "Info@Example.nl", p.Email)
}

func TestNewCompanyProfile_OptionalFieldsMayBeEmpty(t *testing.T) {
	_, err := settings.NewCompanyProfile("tenant-1", settings.CompanyProfileInput{LegalName: "Only A Name B.V."}, now)
	require.NoError(t, err)
}

func TestNewCompanyProfile_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*settings.CompanyProfileInput)
		want   error
	}{
		{"blank legal name", func(in *settings.CompanyProfileInput) { in.LegalName = "   " }, settings.ErrLegalNameRequired},
		{"malformed email", func(in *settings.CompanyProfileInput) { in.Email = "not-an-email" }, settings.ErrInvalidCompanyEmail},
		{"email with display name", func(in *settings.CompanyProfileInput) { in.Email = "Info <info@example.nl>" }, settings.ErrInvalidCompanyEmail},
		{"VAT without country prefix", func(in *settings.CompanyProfileInput) { in.VATNumber = "123456789B01" }, settings.ErrInvalidVATNumber},
		{"VAT too long", func(in *settings.CompanyProfileInput) { in.VATNumber = "NL12345678901234567" }, settings.ErrInvalidVATNumber},
		{"IBAN bad checksum", func(in *settings.CompanyProfileInput) { in.IBAN = "NL92ABNA0417164300" }, settings.ErrInvalidIBAN},
		{"IBAN wrong shape", func(in *settings.CompanyProfileInput) { in.IBAN = "1234" }, settings.ErrInvalidIBAN},
		{"country not two letters", func(in *settings.CompanyProfileInput) { in.Address.Country = "NLD" }, settings.ErrInvalidCountryCode},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)

			_, err := settings.NewCompanyProfile("tenant-1", in, now)

			assert.True(t, errors.Is(err, tc.want), "got %v, want %v", err, tc.want)
		})
	}
}
