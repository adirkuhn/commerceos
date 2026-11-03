package integrations_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type fakeOriginRepo struct {
	rows    []integrationsDomain.SalesOrderOrigin
	saved   []integrationsDomain.SalesOrderOrigin
	findErr error
}

func (f *fakeOriginRepo) Save(_ context.Context, o *integrationsDomain.SalesOrderOrigin) error {
	f.saved = append(f.saved, *o)
	return nil
}

func (f *fakeOriginRepo) find(match func(integrationsDomain.SalesOrderOrigin) bool) (*integrationsDomain.SalesOrderOrigin, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, r := range f.rows {
		if match(r) {
			c := r
			return &c, nil
		}
	}
	return nil, integrationsDomain.ErrSalesOrderOriginNotFound
}

func (f *fakeOriginRepo) FindBySalesOrderID(_ context.Context, id string) (*integrationsDomain.SalesOrderOrigin, error) {
	return f.find(func(o integrationsDomain.SalesOrderOrigin) bool { return o.SalesOrderID == id })
}

func (f *fakeOriginRepo) ListBySalesOrderIDs(_ context.Context, ids []string) ([]integrationsDomain.SalesOrderOrigin, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	var out []integrationsDomain.SalesOrderOrigin
	for _, r := range f.rows {
		for _, id := range ids {
			if r.SalesOrderID == id {
				out = append(out, r)
			}
		}
	}
	return out, nil
}

func (f *fakeOriginRepo) FindByConnectionExternalID(_ context.Context, conn, ext string) (*integrationsDomain.SalesOrderOrigin, error) {
	return f.find(func(o integrationsDomain.SalesOrderOrigin) bool {
		return o.ConnectionID != nil && *o.ConnectionID == conn && o.ExternalOrderID == ext
	})
}

func (f *fakeOriginRepo) FindByOriginExternalID(_ context.Context, origin integrationsDomain.Origin, ext string) (*integrationsDomain.SalesOrderOrigin, error) {
	return f.find(func(o integrationsDomain.SalesOrderOrigin) bool {
		return o.ConnectionID == nil && o.Origin == origin && o.ExternalOrderID == ext
	})
}

func sp(s string) *string { return &s }

func TestSalesOrderOrigins_Record_validatesThenSaves(t *testing.T) {
	repo := &fakeOriginRepo{}
	origins := appintegrations.NewSalesOrderOrigins(repo)

	err := origins.Record(context.Background(), integrationsDomain.SalesOrderOriginInput{
		TenantID: "t1", SalesOrderID: "so-1", Origin: integrationsDomain.OriginWooCommerce, ConnectionID: sp("conn-1"),
		ExternalOrderID: "26217", ExternalKey: "wc_order_k", CreatedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
	assert.Equal(t, "so-1", repo.saved[0].SalesOrderID)

	err = origins.Record(context.Background(), integrationsDomain.SalesOrderOriginInput{SalesOrderID: "so-2", Origin: integrationsDomain.OriginWooCommerce})
	assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginInvalid)
	assert.Len(t, repo.saved, 1, "an invalid origin is never saved")
}

func TestSalesOrderOrigins_FindImported_findsAShopOrderThroughItsConnection(t *testing.T) {
	repo := &fakeOriginRepo{rows: []integrationsDomain.SalesOrderOrigin{
		{SalesOrderID: "so-a", Origin: integrationsDomain.OriginWooCommerce, ConnectionID: sp("shop-a"), ExternalOrderID: "1001"},
		{SalesOrderID: "so-b", Origin: integrationsDomain.OriginWooCommerce, ConnectionID: sp("shop-b"), ExternalOrderID: "1001"},
	}}

	got, err := appintegrations.NewSalesOrderOrigins(repo).FindImported(context.Background(), integrationsDomain.OriginWooCommerce, "shop-b", "1001")

	require.NoError(t, err)
	assert.Equal(t, "so-b", got.SalesOrderID)
	assert.Empty(t, repo.saved)
}

// An order the backfill could not map (several connections of that
// provider) is found by its shop's order id and takes the connection it
// was re-imported through -- never imported a second time.
func TestSalesOrderOrigins_FindImported_claimsAnUnmappedShopOrder(t *testing.T) {
	repo := &fakeOriginRepo{rows: []integrationsDomain.SalesOrderOrigin{
		{SalesOrderID: "so-1", Origin: integrationsDomain.OriginBol, ExternalOrderID: "B-1"},
	}}

	got, err := appintegrations.NewSalesOrderOrigins(repo).FindImported(context.Background(), integrationsDomain.OriginBol, "bol-conn", "B-1")

	require.NoError(t, err)
	assert.Equal(t, "so-1", got.SalesOrderID)
	assert.Equal(t, "bol-conn", *got.ConnectionID)
	require.Len(t, repo.saved, 1)
	assert.Equal(t, "bol-conn", *repo.saved[0].ConnectionID)
}

func TestSalesOrderOrigins_FindImported_unknownOrderIsNotFound(t *testing.T) {
	repo := &fakeOriginRepo{rows: []integrationsDomain.SalesOrderOrigin{
		{SalesOrderID: "so-1", Origin: integrationsDomain.OriginWooCommerce, ExternalOrderID: "1001"},
	}}

	_, err := appintegrations.NewSalesOrderOrigins(repo).FindImported(context.Background(), integrationsDomain.OriginBol, "bol-conn", "1001")

	assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginNotFound, "another origin's order is not this one")
	assert.Empty(t, repo.saved)
}

func TestSalesOrderOrigins_FindImported_aLookupFailureIsReturned(t *testing.T) {
	boom := errors.New("db down")
	_, err := appintegrations.NewSalesOrderOrigins(&fakeOriginRepo{findErr: boom}).FindImported(context.Background(), integrationsDomain.OriginWooCommerce, "c", "1")
	assert.ErrorIs(t, err, boom)
}

func TestSalesOrderOrigins_SetExternalKey_changesOnlyTheKey(t *testing.T) {
	repo := &fakeOriginRepo{rows: []integrationsDomain.SalesOrderOrigin{
		{SalesOrderID: "so-1", Origin: integrationsDomain.OriginManual},
	}}

	require.NoError(t, appintegrations.NewSalesOrderOrigins(repo).SetExternalKey(context.Background(), "so-1", "key-2"))

	require.Len(t, repo.saved, 1)
	assert.Equal(t, "key-2", repo.saved[0].ExternalKey)
	assert.Equal(t, integrationsDomain.OriginManual, repo.saved[0].Origin)

	err := appintegrations.NewSalesOrderOrigins(repo).SetExternalKey(context.Background(), "nope", "k")
	assert.ErrorIs(t, err, integrationsDomain.ErrSalesOrderOriginNotFound)
}
