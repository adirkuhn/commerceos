package theme_test

import (
	"context"
	"fmt"
	"io"
	"sort"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// newMockGormDB allows up to n tenant transactions.
func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		mock.ExpectRollback()
	}
	return gormDB
}

func tenantCtx() context.Context { return tenant.WithTenantID(context.Background(), "tenant-1") }

// ── fakes ───────────────────────────────────────────────────────────────────

type fakeChannels struct {
	byID map[string]channelsDomain.SalesChannel
}

func (f *fakeChannels) FindByID(_ context.Context, id string) (*channelsDomain.SalesChannel, error) {
	c, ok := f.byID[id]
	if !ok {
		return nil, channelsDomain.ErrSalesChannelNotFound
	}
	return &c, nil
}

func themedChannel(id string) channelsDomain.SalesChannel {
	return channelsDomain.SalesChannel{ID: id, TenantID: "tenant-1", Type: channelsDomain.ChannelTypeInternal, StorefrontMode: channelsDomain.StorefrontModeTheme}
}

type fakeThemes struct {
	byID    map[string]theme.Theme
	nextID  int
	deleted []string
}

func (f *fakeThemes) Create(_ context.Context, t *theme.Theme) error {
	f.nextID++
	t.ID = fmt.Sprintf("theme-%d", f.nextID)
	f.byID[t.ID] = *t
	return nil
}

func (f *fakeThemes) FindByID(_ context.Context, id string) (*theme.Theme, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, theme.ErrThemeNotFound
	}
	return &t, nil
}

func (f *fakeThemes) FindByIDForUpdate(ctx context.Context, id string) (*theme.Theme, error) {
	return f.FindByID(ctx, id)
}

func (f *fakeThemes) FindLiveByChannel(_ context.Context, channelID string) (*theme.Theme, error) {
	for _, t := range f.byID {
		if t.SalesChannelID == channelID && t.IsLive {
			return &t, nil
		}
	}
	return nil, nil
}

func (f *fakeThemes) ListByChannel(_ context.Context, channelID string, limit, offset int) ([]theme.Theme, int64, error) {
	var out []theme.Theme
	for _, t := range f.byID {
		if t.SalesChannelID == channelID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	total := int64(len(out))
	if offset > len(out) {
		offset = len(out)
	}
	out = out[offset:]
	if limit < len(out) {
		out = out[:limit]
	}
	return out, total, nil
}

func (f *fakeThemes) Update(_ context.Context, t *theme.Theme) error {
	f.byID[t.ID] = *t
	return nil
}

func (f *fakeThemes) ClearLive(_ context.Context, channelID, except string) error {
	for id, t := range f.byID {
		if t.SalesChannelID == channelID && id != except {
			t.IsLive = false
			f.byID[id] = t
		}
	}
	return nil
}

func (f *fakeThemes) Delete(_ context.Context, id string) error {
	delete(f.byID, id)
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeVersions struct {
	byID   map[string]theme.Version
	nextID int
}

func (f *fakeVersions) Create(_ context.Context, v *theme.Version) error {
	f.nextID++
	v.ID = fmt.Sprintf("v-%d", f.nextID)
	f.byID[v.ID] = *v
	return nil
}

func (f *fakeVersions) FindByID(_ context.Context, id string) (*theme.Version, error) {
	v, ok := f.byID[id]
	if !ok {
		return nil, theme.ErrVersionNotFound
	}
	return &v, nil
}

func (f *fakeVersions) FindDraft(_ context.Context, themeID string, _ bool) (*theme.Version, error) {
	for _, v := range f.byID {
		if v.ThemeID == themeID && v.IsDraft() {
			return &v, nil
		}
	}
	return nil, nil
}

func (f *fakeVersions) NextNumber(_ context.Context, themeID string) (int, error) {
	n := 0
	for _, v := range f.byID {
		if v.ThemeID == themeID && v.Number > n {
			n = v.Number
		}
	}
	return n + 1, nil
}

func (f *fakeVersions) Update(_ context.Context, v *theme.Version) error {
	if old, ok := f.byID[v.ID]; ok && !old.IsDraft() {
		return fmt.Errorf("sealed version %s updated", v.ID)
	}
	f.byID[v.ID] = *v
	return nil
}

func (f *fakeVersions) ListByTheme(_ context.Context, themeID string, limit, offset int) ([]theme.Version, int64, error) {
	var out []theme.Version
	for _, v := range f.byID {
		if v.ThemeID == themeID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number > out[j].Number })
	total := int64(len(out))
	if offset > len(out) {
		offset = len(out)
	}
	out = out[offset:]
	if limit < len(out) {
		out = out[:limit]
	}
	return out, total, nil
}

type fakeDocs struct {
	byID   map[string]theme.Document
	nextID int
}

func (f *fakeDocs) Create(_ context.Context, d *theme.Document) error {
	f.nextID++
	d.ID = fmt.Sprintf("doc-%d", f.nextID)
	f.byID[d.ID] = *d
	return nil
}

func (f *fakeDocs) CopyAll(ctx context.Context, from, to string) error {
	for _, d := range f.forVersion(from) {
		d.VersionID = to
		if err := f.Create(ctx, &d); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeDocs) forVersion(versionID string) []theme.Document {
	var out []theme.Document
	for _, d := range f.byID {
		if d.VersionID == versionID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Kind+theme.DocumentKind(out[i].Key) < out[j].Kind+theme.DocumentKind(out[j].Key)
	})
	return out
}

func (f *fakeDocs) Find(_ context.Context, versionID string, kind theme.DocumentKind, key string) (*theme.Document, error) {
	for _, d := range f.byID {
		if d.VersionID == versionID && d.Kind == kind && d.Key == key {
			return &d, nil
		}
	}
	return nil, theme.ErrDocumentNotFound
}

func (f *fakeDocs) List(_ context.Context, versionID string) ([]theme.Document, error) {
	out := f.forVersion(versionID)
	for i := range out {
		out[i].Root = theme.Node{}
	}
	return out, nil
}

func (f *fakeDocs) Update(_ context.Context, d *theme.Document) error {
	f.byID[d.ID] = *d
	return nil
}

func (f *fakeDocs) Delete(_ context.Context, id string) error {
	delete(f.byID, id)
	return nil
}

type fakeSessions struct{ byHash map[string]theme.Session }

func (f *fakeSessions) Create(_ context.Context, s *theme.Session) error {
	s.ID = "session-" + s.TokenHash[:6]
	f.byHash[s.TokenHash] = *s
	return nil
}

func (f *fakeSessions) FindByTokenHash(_ context.Context, h string) (*theme.Session, error) {
	s, ok := f.byHash[h]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

type fakeAssets struct {
	byID   map[string]theme.Asset
	nextID int
}

func (f *fakeAssets) Create(_ context.Context, a *theme.Asset) error {
	f.nextID++
	a.ID = fmt.Sprintf("00000000-0000-4000-8000-%012d", f.nextID)
	f.byID[a.ID] = *a
	return nil
}

func (f *fakeAssets) FindByID(_ context.Context, id string) (*theme.Asset, error) {
	a, ok := f.byID[id]
	if !ok {
		return nil, theme.ErrAssetNotFound
	}
	return &a, nil
}

func (f *fakeAssets) ListActive(_ context.Context, limit, offset int) ([]theme.Asset, int64, error) {
	var out []theme.Asset
	for _, a := range f.byID {
		if a.DeletedAt == nil {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	total := int64(len(out))
	if offset > len(out) {
		offset = len(out)
	}
	out = out[offset:]
	if limit < len(out) {
		out = out[:limit]
	}
	return out, total, nil
}

func (f *fakeAssets) FindByIDs(_ context.Context, ids []string) ([]theme.Asset, error) {
	var out []theme.Asset
	for _, id := range ids {
		if a, ok := f.byID[id]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeAssets) Update(_ context.Context, a *theme.Asset) error {
	f.byID[a.ID] = *a
	return nil
}

type fakeStorage struct {
	saved []string
	err   error
}

func (f *fakeStorage) Save(_ context.Context, tenantID, filename string, data io.Reader) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	b, _ := io.ReadAll(data)
	f.saved = append(f.saved, filename)
	return fmt.Sprintf("https://cdn.test/%s/%d-%s", tenantID, len(b), filename), nil
}

type fakeOutbox struct{ subjects []string }

func (f *fakeOutbox) Enqueue(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
	f.subjects = append(f.subjects, subject)
	return nil
}

// ── world ───────────────────────────────────────────────────────────────────

// world wires every use case over one shared set of fakes, so tests can
// drive a realistic sequence (create → edit → publish → rollback).
type world struct {
	channels *fakeChannels
	themes   *fakeThemes
	versions *fakeVersions
	docs     *fakeDocs
	sessions *fakeSessions
	outbox   *fakeOutbox
	assets   *fakeAssets
	storage  *fakeStorage
	deps     apptheme.Deps
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{
		channels: &fakeChannels{byID: map[string]channelsDomain.SalesChannel{"chan-1": themedChannel("chan-1")}},
		themes:   &fakeThemes{byID: map[string]theme.Theme{}},
		versions: &fakeVersions{byID: map[string]theme.Version{}},
		docs:     &fakeDocs{byID: map[string]theme.Document{}},
		sessions: &fakeSessions{byHash: map[string]theme.Session{}},
		outbox:   &fakeOutbox{},
		assets:   &fakeAssets{byID: map[string]theme.Asset{}},
		storage:  &fakeStorage{},
	}
	catalog, err := theme.NewCatalog(testCatalogRules(t))
	require.NoError(t, err)
	tokens := 0
	w.deps = apptheme.Deps{
		DB: newMockGormDB(t, 20), Channels: w.channels, Themes: w.themes, Versions: w.versions, Docs: w.docs,
		Sessions: w.sessions, Outbox: w.outbox, Catalog: catalog, Assets: w.assets, Storage: w.storage,
		NewToken: func() (string, error) { tokens++; return fmt.Sprintf("token-%d", tokens), nil },
		Now:      func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) },
	}
	return w
}

func (w *world) createTheme(t *testing.T) theme.Theme {
	t.Helper()
	th, err := apptheme.NewCreateThemeUseCase(w.deps).Execute(tenantCtx(), apptheme.CreateThemeInput{
		SalesChannelID: "chan-1", Name: "Autumn", PresetKey: theme.PresetDefault, UserID: "user-1",
	})
	require.NoError(t, err)
	return *th
}

func (w *world) draft(t *testing.T, themeID string) theme.Version {
	t.Helper()
	v, err := w.versions.FindDraft(context.Background(), themeID, false)
	require.NoError(t, err)
	require.NotNil(t, v)
	return *v
}

// testCatalogRules: strict rules for the types these tests build trees
// from, plus a permissive rule for every other type the default preset
// uses (so preset themes stay publishable in these tests).
func testCatalogRules(t *testing.T) []theme.ComponentRule {
	t.Helper()
	rules := []theme.ComponentRule{
		{Type: "section", Container: true, AllowedParents: []string{theme.RootType}},
		{Type: "heading"},
		{Type: "image"},
	}
	known := map[string]bool{"section": true, "heading": true, "image": true}
	docs, err := theme.PresetDocuments(theme.PresetDefault)
	require.NoError(t, err)
	var visit func(n theme.Node)
	visit = func(n theme.Node) {
		if n.Type != theme.RootType && !known[n.Type] {
			known[n.Type] = true
			rules = append(rules, theme.ComponentRule{Type: n.Type, Container: true})
		}
		for _, c := range n.Children {
			visit(c)
		}
	}
	for _, d := range docs {
		visit(d.Root)
	}
	return rules
}
