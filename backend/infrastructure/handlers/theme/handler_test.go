package theme_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apptheme "github.com/adirkuhn/commerceos/application/theme"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	"github.com/adirkuhn/commerceos/domain/locale"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/domain/theme"
	themehandler "github.com/adirkuhn/commerceos/infrastructure/handlers/theme"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	jwtSecret = "test-secret"
	themeID   = "11111111-1111-4111-8111-111111111111"
	channelID = "22222222-2222-4222-8222-222222222222"
	versionID = "33333333-3333-4333-8333-333333333333"
)

var testNow = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func sampleTheme() theme.Theme {
	return theme.Theme{ID: themeID, TenantID: "tenant-1", SalesChannelID: channelID, Name: "Autumn", PresetKey: "default", CreatedAt: testNow, UpdatedAt: testNow}
}

func sampleVersion(state theme.VersionState) theme.Version {
	return theme.Version{ID: versionID, ThemeID: themeID, Number: 1, State: state, Revision: 3, Settings: theme.DefaultSettings(), CreatedAt: testNow}
}

func sampleDoc() theme.Document {
	return theme.Document{Kind: theme.KindTemplate, Key: "home", Root: theme.Node{ID: "root", Type: theme.RootType, Children: []theme.Node{
		{ID: "s1", Type: "section", Style: theme.NodeStyle{Base: map[string]string{"padding_y": "xl"}}},
	}}, Translations: map[locale.Locale]theme.DocumentTexts{locale.Dutch: {SEO: theme.SEO{Title: "Winkel"}}}}
}

// stub is one configurable fake satisfying every use-case port the handler
// needs; each test sets only what it exercises.
type stub struct {
	err            error
	gotCtx         context.Context
	gotSave        apptheme.SaveDraftDocumentInput
	gotPublish     apptheme.PublishThemeInput
	gotSession     apptheme.CreateSessionInput
	gotRevision    int
	gotPreviewCh   string
	session        *theme.Session
	sessionErr     error
	key            apikeysDomain.ResolvedAPIKey
	keyErr         error
	gotUpload      apptheme.UploadAssetInput
	gotUploadBytes []byte
	appearance     *settingsDomain.Appearance
	appearanceErr  error
	gotAppearCtx   context.Context
}

const assetID = "44444444-4444-4444-8444-444444444444"

func sampleAsset() theme.Asset {
	return theme.Asset{ID: assetID, URL: "https://cdn.test/a.png", Filename: "a.png", ContentType: "image/png", Bytes: 10, Width: 4, Height: 3, Alt: "A", CreatedAt: testNow}
}

func sampleAssets() map[string]theme.Asset { return map[string]theme.Asset{assetID: sampleAsset()} }

func (s *stub) list(ctx context.Context, _ string, _, _ int) ([]theme.Theme, int64, error) {
	s.gotCtx = ctx
	return []theme.Theme{sampleTheme()}, 7, s.err
}

func newRouter(t *testing.T, s *stub) *gin.Engine {
	t.Helper()
	uc := themehandler.UseCases{
		ListThemes: listFunc(s.list),
		CreateTheme: createFunc(func(ctx context.Context, in apptheme.CreateThemeInput) (*theme.Theme, error) {
			s.gotCtx = ctx
			th := sampleTheme()
			th.Name = in.Name
			return &th, s.err
		}),
		GetTheme: getFunc(func(context.Context, string) (*apptheme.ThemeDetail, error) {
			v := sampleVersion(theme.VersionStateDraft)
			return &apptheme.ThemeDetail{Theme: sampleTheme(), Draft: &v}, s.err
		}),
		RenameTheme: renameFunc(func(_ context.Context, _, name string) (*theme.Theme, error) {
			th := sampleTheme()
			th.Name = name
			return &th, s.err
		}),
		DeleteTheme: deleteFunc(func(context.Context, string) error { return s.err }),
		ListVersions: versionsFunc(func(context.Context, string, int, int) ([]theme.Version, int64, error) {
			return []theme.Version{sampleVersion(theme.VersionStateSealed)}, 1, s.err
		}),
		PublishTheme: publishFunc(func(_ context.Context, in apptheme.PublishThemeInput) (*theme.Version, error) {
			s.gotPublish = in
			v := sampleVersion(theme.VersionStateSealed)
			return &v, s.err
		}),
		RollbackTheme: rollbackFunc(func(context.Context, string, string) (*theme.Theme, error) { th := sampleTheme(); return &th, s.err }),
		CreateSession: sessionFunc(func(_ context.Context, in apptheme.CreateSessionInput) (*apptheme.SessionToken, error) {
			s.gotSession = in
			return &apptheme.SessionToken{Token: "raw-token", ExpiresAt: testNow}, s.err
		}),
		ResolveSession: resolveFunc(func(_ context.Context, token string, purpose theme.SessionPurpose) (*theme.Session, error) {
			if s.session == nil || token != "good" || s.session.Purpose != purpose {
				return nil, theme.ErrSessionInvalid
			}
			return s.session, s.sessionErr
		}),
		OpenDraft: openFunc(func(ctx context.Context, _, _ string) (*apptheme.VersionView, error) {
			s.gotCtx = ctx
			return &apptheme.VersionView{Theme: sampleTheme(), Version: sampleVersion(theme.VersionStateDraft), Documents: []theme.Document{{Kind: theme.KindTemplate, Key: "home"}}}, s.err
		}),
		SaveDraftSettings: settingsFunc(func(context.Context, apptheme.SaveDraftSettingsInput) (*theme.Version, error) {
			v := sampleVersion(theme.VersionStateDraft)
			return &v, s.err
		}),
		GetDraftDocument: getDocFunc(func(context.Context, string, theme.DocumentKind, string) (*apptheme.DocumentView, error) {
			return &apptheme.DocumentView{Document: sampleDoc(), Assets: sampleAssets()}, s.err
		}),
		SaveDraftDocument: saveDocFunc(func(_ context.Context, in apptheme.SaveDraftDocumentInput) (*apptheme.DraftWrite, error) {
			s.gotSave = in
			return &apptheme.DraftWrite{Document: sampleDoc(), Revision: 4}, s.err
		}),
		CreatePage: createPageFunc(func(_ context.Context, in apptheme.CreatePageInput) (*apptheme.DraftWrite, error) {
			return &apptheme.DraftWrite{Document: theme.Document{Kind: theme.KindPage, Key: in.Key, Title: in.Title, Root: theme.EmptyRoot()}, Revision: 2}, s.err
		}),
		DeletePage: deletePageFunc(func(_ context.Context, _ string, rev int, _ string) (int, error) {
			s.gotRevision = rev
			return rev + 1, s.err
		}),
		GetLiveTheme: liveFunc(func(ctx context.Context, _ string) (*apptheme.VersionView, error) {
			s.gotCtx = ctx
			return &apptheme.VersionView{Theme: sampleTheme(), Version: sampleVersion(theme.VersionStateSealed)}, s.err
		}),
		GetPublishedDocument: publishedDocFunc(func(context.Context, string, string, theme.DocumentKind, string) (*apptheme.DocumentView, error) {
			return &apptheme.DocumentView{Document: sampleDoc(), Assets: sampleAssets()}, s.err
		}),
		GetPreviewTheme: previewFunc(func(_ context.Context, ch, _ string) (*apptheme.VersionView, error) {
			s.gotPreviewCh = ch
			return &apptheme.VersionView{Theme: sampleTheme(), Version: sampleVersion(theme.VersionStateDraft)}, s.err
		}),
		GetPreviewDocument: previewDocFunc(func(context.Context, string, string, theme.DocumentKind, string) (*apptheme.DocumentView, error) {
			return &apptheme.DocumentView{Document: sampleDoc()}, s.err
		}),
		ListAssets: listAssetsFunc(func(context.Context, int, int) ([]theme.Asset, int64, error) {
			return []theme.Asset{sampleAsset()}, 1, s.err
		}),
		UploadAsset: uploadAssetFunc(func(_ context.Context, in apptheme.UploadAssetInput) (*theme.Asset, error) {
			s.gotUpload = in
			if in.Data != nil {
				s.gotUploadBytes, _ = io.ReadAll(in.Data)
			}
			a := sampleAsset()
			return &a, s.err
		}),
		UpdateAsset: updateAssetFunc(func(_ context.Context, _, alt string) (*theme.Asset, error) {
			a := sampleAsset()
			a.Alt = alt
			return &a, s.err
		}),
		DeleteAsset: deleteFunc(func(context.Context, string) error { return s.err }),
		GetAppearance: appearanceFunc(func(ctx context.Context) (*settingsDomain.Appearance, error) {
			s.gotAppearCtx = ctx
			if s.appearanceErr != nil {
				return nil, s.appearanceErr
			}
			if s.appearance != nil {
				return s.appearance, nil
			}
			a := settingsDomain.DefaultAppearance(tenant.IDFromContext(ctx))
			return &a, nil
		}),
		ResolveStorefrontKey: keyFunc(func(_ context.Context, k string) (apikeysDomain.ResolvedAPIKey, error) {
			if k != "shop-key" {
				return apikeysDomain.ResolvedAPIKey{}, errors.New("bad key")
			}
			return s.key, s.keyErr
		}),
	}
	r := gin.New()
	h := themehandler.NewHandler(uc)
	admin := r.Group("/api/v1")
	admin.Use(middleware.JWT([]byte(jwtSecret)))
	themehandler.RegisterAdmin(admin, h)
	themehandler.RegisterPublic(r.Group("/api/v1"), h)
	return r
}

func jwtFor(t *testing.T, role middleware.Role) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
		UserID: "user-1", TenantID: "tenant-1", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	s, err := tok.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return s
}

func do(r *gin.Engine, method, path, body, bearer string, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// ── admin ──

func TestAdmin_requiresJWT(t *testing.T) {
	w := do(newRouter(t, &stub{}), http.MethodGet, "/api/v1/theme/themes?sales_channel_id="+channelID, "", "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdmin_listThemes(t *testing.T) {
	s := &stub{}
	w := do(newRouter(t, s), http.MethodGet, "/api/v1/theme/themes?sales_channel_id="+channelID+"&page=2&per_page=5", "", jwtFor(t, middleware.RoleCustomerUser))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	body := decode(t, w)
	assert.EqualValues(t, 7, body["total"])
	assert.Equal(t, "Autumn", body["data"].([]any)[0].(map[string]any)["name"])
	assert.Equal(t, "tenant-1", tenant.IDFromContext(s.gotCtx))
}

func TestAdmin_readOnlyUserCannotWrite(t *testing.T) {
	w := do(newRouter(t, &stub{}), http.MethodPost, "/api/v1/theme/themes", `{"sales_channel_id":"`+channelID+`","name":"X"}`, jwtFor(t, middleware.RoleCustomerUser))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdmin_createTheme(t *testing.T) {
	w := do(newRouter(t, &stub{}), http.MethodPost, "/api/v1/theme/themes", `{"sales_channel_id":"`+channelID+`","name":"Winter"}`, jwtFor(t, middleware.RoleCustomerAdmin))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "Winter", decode(t, w)["data"].(map[string]any)["name"])
}

func TestAdmin_errorMapping(t *testing.T) {
	cases := []struct {
		err    error
		method string
		path   string
		body   string
		status int
	}{
		{theme.ErrInvalidThemeName, http.MethodPost, "/api/v1/theme/themes", `{"sales_channel_id":"` + channelID + `","name":"X"}`, http.StatusUnprocessableEntity},
		{theme.ErrStorefrontNotEnabled, http.MethodPost, "/api/v1/theme/themes", `{"sales_channel_id":"` + channelID + `","name":"X"}`, http.StatusConflict},
		{theme.ErrThemeNotFound, http.MethodGet, "/api/v1/theme/themes/" + themeID, "", http.StatusNotFound},
		{theme.ErrThemeIsLive, http.MethodDelete, "/api/v1/theme/themes/" + themeID, "", http.StatusConflict},
		{theme.ErrRevisionConflict, http.MethodPost, "/api/v1/theme/themes/" + themeID + "/publish", `{"expected_revision":1}`, http.StatusConflict},
		{theme.ErrVersionNotSealed, http.MethodPost, "/api/v1/theme/themes/" + themeID + "/rollback", `{"version_id":"` + versionID + `"}`, http.StatusUnprocessableEntity},
		{errors.New("db down"), http.MethodGet, "/api/v1/theme/themes/" + themeID, "", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		w := do(newRouter(t, &stub{err: tc.err}), tc.method, tc.path, tc.body, jwtFor(t, middleware.RoleCustomerOwner))
		assert.Equal(t, tc.status, w.Code, "%v -> %s", tc.err, w.Body.String())
	}
}

func TestAdmin_publishPassesRevisionAndNote(t *testing.T) {
	s := &stub{}
	w := do(newRouter(t, s), http.MethodPost, "/api/v1/theme/themes/"+themeID+"/publish", `{"expected_revision":3,"note":"launch"}`, jwtFor(t, middleware.RoleCustomerOwner))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, apptheme.PublishThemeInput{ThemeID: themeID, ExpectedRevision: 3, Note: "launch"}, s.gotPublish)
	assert.Equal(t, "sealed", decode(t, w)["data"].(map[string]any)["state"])
}

func TestAdmin_createSessionUsesCallerIdentity(t *testing.T) {
	s := &stub{}
	w := do(newRouter(t, s), http.MethodPost, "/api/v1/theme/themes/"+themeID+"/sessions", `{"purpose":"edit"}`, jwtFor(t, middleware.RoleCustomerAdmin))

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "raw-token", decode(t, w)["data"].(map[string]any)["token"])
	assert.Equal(t, apptheme.CreateSessionInput{ThemeID: themeID, UserID: "user-1", Role: "customer_admin", Purpose: theme.PurposeEdit}, s.gotSession)
}

// The merchant's language goes into the session (never the editor URL).
func TestAdmin_createSessionCarriesTheMerchantLanguage(t *testing.T) {
	s := &stub{}
	w := do(newRouter(t, s), http.MethodPost, "/api/v1/theme/themes/"+themeID+"/sessions", `{"purpose":"edit","locale":"pt-BR"}`, jwtFor(t, middleware.RoleCustomerAdmin))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, locale.Portuguese, s.gotSession.Locale)
}

func TestAdmin_getThemeIncludesDraftRevision(t *testing.T) {
	w := do(newRouter(t, &stub{}), http.MethodGet, "/api/v1/theme/themes/"+themeID, "", jwtFor(t, middleware.RoleCustomerUser))

	require.Equal(t, http.StatusOK, w.Code)
	draft := decode(t, w)["data"].(map[string]any)["draft"].(map[string]any)
	assert.EqualValues(t, 3, draft["revision"])
}
