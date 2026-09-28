package collections_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appcollections "github.com/adirkuhn/commerceos/application/collections"
	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func mockDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 20; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		mock.ExpectRollback()
	}
	return db
}

type fakeRepo struct {
	byID     map[string]collections.Collection
	manual   map[string][]string
	resolved []string
	n        int
}

func (f *fakeRepo) Create(_ context.Context, c *collections.Collection) error {
	for _, e := range f.byID {
		if e.Slug == c.Slug {
			return collections.ErrSlugTaken
		}
	}
	f.n++
	c.ID = fmt.Sprintf("c%d", f.n)
	f.byID[c.ID] = *c
	return nil
}
func (f *fakeRepo) FindByID(_ context.Context, id string) (*collections.Collection, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, collections.ErrCollectionNotFound
	}
	return &c, nil
}
func (f *fakeRepo) FindBySlug(_ context.Context, slug string) (*collections.Collection, error) {
	for _, c := range f.byID {
		if c.Slug == slug {
			return &c, nil
		}
	}
	return nil, collections.ErrCollectionNotFound
}
func (f *fakeRepo) List(_ context.Context, published bool, _, _ int) ([]collections.Collection, int64, error) {
	var out []collections.Collection
	for _, c := range f.byID {
		if !published || c.Published {
			out = append(out, c)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeRepo) Update(_ context.Context, c *collections.Collection) error {
	f.byID[c.ID] = *c
	return nil
}
func (f *fakeRepo) Delete(_ context.Context, id string) error { delete(f.byID, id); return nil }
func (f *fakeRepo) ManualProducts(_ context.Context, id string) ([]string, error) {
	return f.manual[id], nil
}
func (f *fakeRepo) SetManualProducts(_ context.Context, id string, ids []string) error {
	f.manual[id] = ids
	return nil
}
func (f *fakeRepo) ResolveProductIDs(_ context.Context, _ collections.Collection, limit int) ([]string, error) {
	if len(f.resolved) > limit {
		return f.resolved[:limit], nil
	}
	return f.resolved, nil
}

type fakeOutbox struct{ subjects []string }

func (f *fakeOutbox) Enqueue(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
	f.subjects = append(f.subjects, subject)
	return nil
}

func setup(t *testing.T) (appcollections.Deps, *fakeRepo, *fakeOutbox) {
	repo := &fakeRepo{byID: map[string]collections.Collection{}, manual: map[string][]string{}}
	out := &fakeOutbox{}
	return appcollections.Deps{DB: mockDB(t), Repo: repo, Outbox: out, Now: func() time.Time { return time.Unix(0, 0) }}, repo, out
}

var ctx = tenant.WithTenantID(context.Background(), "tenant-1")

func manualDraft(slug string) collections.Draft {
	return collections.Draft{Name: "Picks", Slug: slug, Type: collections.TypeManual, Sort: collections.SortManual}
}

func TestCreateAndUpdate(t *testing.T) {
	d, repo, out := setup(t)

	c, err := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))
	require.NoError(t, err)
	assert.Equal(t, "tenant-1", c.TenantID)

	_, err = appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))
	assert.ErrorIs(t, err, collections.ErrSlugTaken)
	_, err = appcollections.NewCreateCollectionUseCase(d).Execute(ctx, collections.Draft{Name: "", Slug: "x", Type: collections.TypeManual, Sort: collections.SortManual})
	assert.ErrorIs(t, err, collections.ErrInvalidCollection)

	upd := manualDraft("picks")
	upd.Name = "Top picks"
	got, err := appcollections.NewUpdateCollectionUseCase(d).Execute(ctx, c.ID, upd)
	require.NoError(t, err)
	assert.Equal(t, "Top picks", got.Name)
	assert.Equal(t, "Top picks", repo.byID[c.ID].Name)
	assert.Equal(t, []string{"collections.collection.changed", "collections.collection.changed"}, out.subjects)
}

func TestSwitchingToSmartClearsManualProducts(t *testing.T) {
	d, repo, _ := setup(t)
	c, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))
	require.NoError(t, appcollections.NewSetCollectionProductsUseCase(d).Execute(ctx, c.ID, []string{"p1", "p2"}))

	smart := collections.Draft{Name: "Sale", Slug: "picks", Type: collections.TypeSmart, Match: collections.MatchAll,
		Sort: collections.SortNewest, Rules: []collections.Rule{{Field: collections.FieldOnSale}}}
	_, err := appcollections.NewUpdateCollectionUseCase(d).Execute(ctx, c.ID, smart)
	require.NoError(t, err)
	assert.Empty(t, repo.manual[c.ID])
}

func TestSetCollectionProducts(t *testing.T) {
	d, repo, _ := setup(t)
	c, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))
	uc := appcollections.NewSetCollectionProductsUseCase(d)

	require.NoError(t, uc.Execute(ctx, c.ID, []string{"p2", "p1"}))
	assert.Equal(t, []string{"p2", "p1"}, repo.manual[c.ID])
	assert.ErrorIs(t, uc.Execute(ctx, c.ID, []string{"p1", "p1"}), collections.ErrInvalidCollection)

	s, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, collections.Draft{Name: "S", Slug: "s", Type: collections.TypeSmart,
		Match: collections.MatchAll, Sort: collections.SortNewest, Rules: []collections.Rule{{Field: collections.FieldOnSale}}})
	assert.ErrorIs(t, uc.Execute(ctx, s.ID, []string{"p1"}), collections.ErrNotManual)
}

func TestDelete(t *testing.T) {
	d, repo, out := setup(t)
	c, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))

	require.NoError(t, appcollections.NewDeleteCollectionUseCase(d).Execute(ctx, c.ID))
	assert.Empty(t, repo.byID)
	assert.Len(t, out.subjects, 2)
	assert.ErrorIs(t, appcollections.NewDeleteCollectionUseCase(d).Execute(ctx, c.ID), collections.ErrCollectionNotFound)
}

func TestPublishedReads(t *testing.T) {
	d, repo, _ := setup(t)
	draft := manualDraft("hidden")
	hidden, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, draft)
	pub := manualDraft("shown")
	pub.Published = true
	shown, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, pub)
	repo.resolved = []string{"p1", "p2", "p3"}

	_, err := appcollections.NewGetPublishedCollectionUseCase(d).Execute(ctx, "hidden")
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound, "unpublished collections are invisible to shoppers")

	got, err := appcollections.NewGetPublishedCollectionUseCase(d).Execute(ctx, "shown")
	require.NoError(t, err)
	assert.Equal(t, shown.ID, got.ID)

	list, total, err := appcollections.NewListCollectionsUseCase(d).Execute(ctx, true, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, shown.ID, list[0].ID)

	ids, err := appcollections.NewResolveCollectionProductsUseCase(d).Execute(ctx, shown.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"p1", "p2", "p3"}, ids)
	_, err = appcollections.NewResolveCollectionProductsUseCase(d).Execute(ctx, hidden.ID)
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func TestGetCollectionIncludesManualProducts(t *testing.T) {
	d, _, _ := setup(t)
	c, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))
	require.NoError(t, appcollections.NewSetCollectionProductsUseCase(d).Execute(ctx, c.ID, []string{"p1"}))

	got, err := appcollections.NewGetCollectionUseCase(d).Execute(ctx, c.ID)
	require.NoError(t, err)
	assert.Equal(t, []string{"p1"}, got.ProductIDs)

	_, err = appcollections.NewGetCollectionUseCase(d).Execute(ctx, "missing")
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
}

func TestAdminListIncludesUnpublished(t *testing.T) {
	d, _, _ := setup(t)
	_, _ = appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("a"))
	_, total, err := appcollections.NewListCollectionsUseCase(d).Execute(ctx, false, 0, 500)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
}

func TestUpdateErrors(t *testing.T) {
	d, repo, _ := setup(t)
	_, err := appcollections.NewUpdateCollectionUseCase(d).Execute(ctx, "missing", manualDraft("x"))
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)

	c, _ := appcollections.NewCreateCollectionUseCase(d).Execute(ctx, manualDraft("picks"))
	bad := manualDraft("picks")
	bad.Slug = "Not A Slug"
	_, err = appcollections.NewUpdateCollectionUseCase(d).Execute(ctx, c.ID, bad)
	assert.ErrorIs(t, err, collections.ErrInvalidCollection)
	assert.Equal(t, "picks", repo.byID[c.ID].Slug)

	assert.ErrorIs(t, appcollections.NewSetCollectionProductsUseCase(d).Execute(ctx, "missing", nil), collections.ErrCollectionNotFound)
	_, err = appcollections.NewGetPublishedCollectionUseCase(d).Execute(ctx, "nope")
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
	_, err = appcollections.NewResolveCollectionProductsUseCase(d).Execute(ctx, "nope")
	assert.ErrorIs(t, err, collections.ErrCollectionNotFound)
}
