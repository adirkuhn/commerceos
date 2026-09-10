package theme_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	appTheme "github.com/adirkuhn/commerceos/application/theme"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
	themehandler "github.com/adirkuhn/commerceos/infrastructure/handlers/theme"
)

func init() { gin.SetMode(gin.TestMode) }

type stubGetThemePage struct {
	out *themeDomain.ThemePage
	err error

	gotChannelID string
	gotSlug      themeDomain.PageSlug
}

func (s *stubGetThemePage) Execute(_ context.Context, channelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error) {
	s.gotChannelID, s.gotSlug = channelID, slug
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &themeDomain.ThemePage{SalesChannelID: channelID, Slug: slug, CanvasWidth: 1440}, nil
}

type stubSaveThemePage struct {
	out *themeDomain.ThemePage
	err error

	gotInput appTheme.SaveThemePageInput
}

func (s *stubSaveThemePage) Execute(_ context.Context, in appTheme.SaveThemePageInput) (*themeDomain.ThemePage, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &themeDomain.ThemePage{
		SalesChannelID: in.SalesChannelID, Slug: in.Slug, CanvasWidth: in.CanvasWidth, Blocks: in.Blocks, CustomCSS: in.CustomCSS,
	}, nil
}

type stubCreateEditSession struct {
	out *themeDomain.EditSession
	err error

	gotChannelID string
}

func (s *stubCreateEditSession) Execute(_ context.Context, channelID string) (*themeDomain.EditSession, error) {
	s.gotChannelID = channelID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &themeDomain.EditSession{SalesChannelID: channelID, Token: "test-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func newTestHandler(get *stubGetThemePage, save *stubSaveThemePage, editSessions ...*stubCreateEditSession) *themehandler.Handler {
	es := &stubCreateEditSession{}
	if len(editSessions) > 0 {
		es = editSessions[0]
	}
	return themehandler.NewHandler(get, save, es)
}

func newTestRouter(h *themehandler.Handler) *gin.Engine {
	r := gin.New()
	gentheme.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestGetThemePageHandler_returnsDefaultPage(t *testing.T) {
	get := &stubGetThemePage{}
	h := newTestHandler(get, &stubSaveThemePage{})
	r := newTestRouter(h)

	w := doRequest(r, http.MethodGet, "/api/v1/theme/sales-channels/chan-1/pages/home", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp gentheme.ThemePageResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	assert.Equal(t, "chan-1", resp.Data.SalesChannelId)
	assert.Equal(t, gentheme.PageSlug("home"), resp.Data.Slug)
	assert.Equal(t, 1440, resp.Data.CanvasWidth)
	assert.Equal(t, "chan-1", get.gotChannelID)
	assert.Equal(t, themeDomain.PageSlugHome, get.gotSlug)
}

func TestGetThemePageHandler_returnsSavedPage(t *testing.T) {
	get := &stubGetThemePage{out: &themeDomain.ThemePage{
		SalesChannelID: "chan-1",
		Slug:           themeDomain.PageSlugHome,
		CanvasWidth:    1200,
		Blocks: []themeDomain.Block{
			{ID: "b1", Type: themeDomain.BlockTypeHero, Layout: themeDomain.BlockLayout{Desktop: themeDomain.Rect{X: 1, Y: 2, Width: 3, Height: 4}}, Props: map[string]any{"heading": "hi"}},
		},
	}}
	h := newTestHandler(get, &stubSaveThemePage{})
	r := newTestRouter(h)

	w := doRequest(r, http.MethodGet, "/api/v1/theme/sales-channels/chan-1/pages/home", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp gentheme.ThemePageResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	require.Len(t, resp.Data.Blocks, 1)
	assert.Equal(t, "b1", resp.Data.Blocks[0].Id)
	assert.Equal(t, gentheme.BlockType("hero"), resp.Data.Blocks[0].Type)
	assert.Equal(t, 1, resp.Data.Blocks[0].Layout.Desktop.X)
}

func TestGetThemePageHandler_notFoundChannelReturns404(t *testing.T) {
	get := &stubGetThemePage{err: channelsDomain.ErrSalesChannelNotFound}
	h := newTestHandler(get, &stubSaveThemePage{})
	r := newTestRouter(h)

	w := doRequest(r, http.MethodGet, "/api/v1/theme/sales-channels/missing/pages/home", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSaveThemePageHandler_success(t *testing.T) {
	save := &stubSaveThemePage{}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	body := gentheme.SaveThemePageRequest{
		CanvasWidth: 1440,
		Blocks: []gentheme.Block{
			{Id: "b1", Type: gentheme.BlockType("hero"), Layout: gentheme.BlockLayout{Desktop: gentheme.Rect{X: 0, Y: 0, Width: 100, Height: 100}}, Props: map[string]interface{}{}},
		},
	}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/chan-1/pages/home", body)

	require.Equal(t, http.StatusOK, w.Code)
	var resp gentheme.ThemePageResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	require.Len(t, resp.Data.Blocks, 1)
	assert.Equal(t, "chan-1", save.gotInput.SalesChannelID)
	assert.Equal(t, themeDomain.PageSlugHome, save.gotInput.Slug)
	assert.Equal(t, themeDomain.BlockTypeHero, save.gotInput.Blocks[0].Type)
}

func TestSaveThemePageHandler_roundTripsCustomCSS(t *testing.T) {
	save := &stubSaveThemePage{}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	body := gentheme.SaveThemePageRequest{CanvasWidth: 1440, Blocks: []gentheme.Block{}, CustomCss: ".hero { color: red; }"}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/chan-1/pages/home", body)

	require.Equal(t, http.StatusOK, w.Code)
	var resp gentheme.ThemePageResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	assert.Equal(t, ".hero { color: red; }", resp.Data.CustomCss)
	assert.Equal(t, ".hero { color: red; }", save.gotInput.CustomCSS)
}

func TestSaveThemePageHandler_invalidSlugReturns422(t *testing.T) {
	save := &stubSaveThemePage{err: themeDomain.ErrInvalidPageSlug}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	body := gentheme.SaveThemePageRequest{CanvasWidth: 1440, Blocks: []gentheme.Block{}}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/chan-1/pages/home", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSaveThemePageHandler_invalidBlockTypeReturns422(t *testing.T) {
	save := &stubSaveThemePage{err: themeDomain.ErrInvalidBlockType}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	body := gentheme.SaveThemePageRequest{
		CanvasWidth: 1440,
		Blocks: []gentheme.Block{
			{Id: "b1", Type: gentheme.BlockType("not-a-real-type"), Layout: gentheme.BlockLayout{Desktop: gentheme.Rect{}}, Props: map[string]interface{}{}},
		},
	}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/chan-1/pages/home", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSaveThemePageHandler_roundTripsPuckModeAndData(t *testing.T) {
	save := &stubSaveThemePage{}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	puckMode := gentheme.Puck
	puckData := map[string]interface{}{"content": []interface{}{"x"}}
	body := gentheme.SaveThemePageRequest{CanvasWidth: 1440, Blocks: []gentheme.Block{}, EditorMode: &puckMode, PuckData: &puckData}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/chan-1/pages/home", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, themeDomain.EditorModePuck, save.gotInput.EditorMode)
	assert.Equal(t, map[string]any{"content": []any{"x"}}, save.gotInput.PuckData)
}

func TestSaveThemePageHandler_defaultsToFreeformWhenEditorModeOmitted(t *testing.T) {
	save := &stubSaveThemePage{}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	body := gentheme.SaveThemePageRequest{CanvasWidth: 1440, Blocks: []gentheme.Block{}}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/chan-1/pages/home", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, themeDomain.EditorModeFreeform, save.gotInput.EditorMode)
}

func TestSaveThemePageHandler_unknownChannelReturns404(t *testing.T) {
	save := &stubSaveThemePage{err: channelsDomain.ErrSalesChannelNotFound}
	h := newTestHandler(&stubGetThemePage{}, save)
	r := newTestRouter(h)

	body := gentheme.SaveThemePageRequest{CanvasWidth: 1440, Blocks: []gentheme.Block{}}
	w := doRequest(r, http.MethodPut, "/api/v1/theme/sales-channels/missing/pages/home", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateThemeEditSessionHandler_success(t *testing.T) {
	create := &stubCreateEditSession{out: &themeDomain.EditSession{SalesChannelID: "chan-1", Token: "abc123", ExpiresAt: time.Now().Add(time.Hour)}}
	h := newTestHandler(&stubGetThemePage{}, &stubSaveThemePage{}, create)
	r := newTestRouter(h)

	w := doRequest(r, http.MethodPost, "/api/v1/theme/sales-channels/chan-1/edit-sessions", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var resp gentheme.EditSessionResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	assert.Equal(t, "abc123", resp.Data.Token)
	assert.Equal(t, "chan-1", create.gotChannelID)
}

func TestCreateThemeEditSessionHandler_unknownChannelReturns404(t *testing.T) {
	create := &stubCreateEditSession{err: channelsDomain.ErrSalesChannelNotFound}
	h := newTestHandler(&stubGetThemePage{}, &stubSaveThemePage{}, create)
	r := newTestRouter(h)

	w := doRequest(r, http.MethodPost, "/api/v1/theme/sales-channels/missing/edit-sessions", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
