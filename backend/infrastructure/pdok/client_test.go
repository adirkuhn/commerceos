package pdok_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/infrastructure/pdok"
)

func docsResponse(postcode, huisNLT, straatnaam, woonplaats string) string {
	return `{"response":{"numFound":1,"docs":[{"postcode":"` + postcode + `","huis_nlt":"` + huisNLT +
		`","straatnaam":"` + straatnaam + `","woonplaatsnaam":"` + woonplaats + `"}]}}`
}

func TestClient_Lookup_returnsMatchWhenPostcodeAndHouseNumberBothConfirm(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(docsResponse("1035SH", "31", "Thomas Prinslaan", "Amsterdam")))
	}))
	defer srv.Close()

	c := pdok.NewClient(pdok.Config{BaseURL: srv.URL})
	result, err := c.Lookup(context.Background(), "1035SH", "31")

	require.NoError(t, err)
	assert.Equal(t, "Thomas Prinslaan", result.Street)
	assert.Equal(t, "Amsterdam", result.City)
	assert.Contains(t, gotQuery, "fq=type%3Aadres")
}

// TestClient_Lookup_rejectsFuzzyMismatch is a regression test for real,
// observed PDOK behavior: its free-text search always returns *some*
// result, even a completely unrelated address in a different city, for a
// postal code + house number combination that doesn't exist. This must
// never be surfaced as a match.
func TestClient_Lookup_rejectsFuzzyMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Real observed response for a nonexistent "9999ZZ 999" query --
		// same house number, entirely different postcode/city.
		_, _ = w.Write([]byte(docsResponse("3555HS", "999", "Amsterdamsestraatweg", "Utrecht")))
	}))
	defer srv.Close()

	c := pdok.NewClient(pdok.Config{BaseURL: srv.URL})
	_, err := c.Lookup(context.Background(), "9999ZZ", "999")

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrAddressNotFound)
}

func TestClient_Lookup_rejectsRightPostcodeWrongHouseNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Real observed behavior: postcode matched, but the fuzzy search
		// substituted a different house number entirely (1 instead of 9999).
		_, _ = w.Write([]byte(docsResponse("1035SH", "1", "Thomas Prinslaan", "Amsterdam")))
	}))
	defer srv.Close()

	c := pdok.NewClient(pdok.Config{BaseURL: srv.URL})
	_, err := c.Lookup(context.Background(), "1035SH", "9999")

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrAddressNotFound)
}

func TestClient_Lookup_isCaseAndSpaceInsensitive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(docsResponse("1035SH", "31A", "Thomas Prinslaan", "Amsterdam")))
	}))
	defer srv.Close()

	c := pdok.NewClient(pdok.Config{BaseURL: srv.URL})
	result, err := c.Lookup(context.Background(), "1035 sh", "31a")

	require.NoError(t, err)
	assert.Equal(t, "Thomas Prinslaan", result.Street)
}

func TestClient_Lookup_emptyDocsReturnsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":{"numFound":0,"docs":[]}}`))
	}))
	defer srv.Close()

	c := pdok.NewClient(pdok.Config{BaseURL: srv.URL})
	_, err := c.Lookup(context.Background(), "1035SH", "31")

	require.Error(t, err)
	assert.ErrorIs(t, err, shippingDomain.ErrAddressNotFound)
}

func TestClient_Lookup_nonOKStatus_returnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := pdok.NewClient(pdok.Config{BaseURL: srv.URL})
	_, err := c.Lookup(context.Background(), "1035SH", "31")

	require.Error(t, err)
	assert.NotErrorIs(t, err, shippingDomain.ErrAddressNotFound, "an upstream failure is distinct from a confirmed non-match")
}
