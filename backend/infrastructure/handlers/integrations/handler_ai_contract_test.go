package integrations_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	integrationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
)

const (
	aiID1 = "11111111-1111-4111-8111-111111111111"
	aiID2 = "22222222-2222-4222-8222-222222222222"
)

// newContractRouter runs requests through the same OpenAPI request
// validator cmd/api mounts on the integrations group, so these tests pin
// the contract (openapi.yaml), not just the handler.
func newContractRouter(t *testing.T, h *integrationshandler.Handler) *gin.Engine {
	t.Helper()
	spec, err := genintegrations.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	integrationshandler.RegisterRoutes(g, h)
	return r
}

var aiAt = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func sampleAIConns() []integrationsDomain.AIConnection {
	return []integrationsDomain.AIConnection{
		{ID: aiID1, Position: 0, Label: "NVIDIA key 1", Enabled: true, Provider: integrationsDomain.AIProviderNVIDIA,
			APIKeyEncrypted: "enc", Model: "meta/llama-3.3-70b-instruct", Status: integrationsDomain.StatusConnected, CreatedAt: aiAt},
		{ID: aiID2, Position: 1, Enabled: false, Provider: integrationsDomain.AIProviderOllama, BaseURL: "http://ollama:11434",
			Status: integrationsDomain.StatusConnected, CreatedAt: aiAt},
	}
}

type stubAIList struct{ err error }

func (s stubAIList) Execute(context.Context) ([]integrationsDomain.AIConnection, error) {
	return sampleAIConns(), s.err
}

type stubAICreate struct {
	got appintegrations.AIConnectionInput
	err error
}

func (s *stubAICreate) Execute(_ context.Context, in appintegrations.AIConnectionInput) (*integrationsDomain.AIConnection, error) {
	s.got = in
	if s.err != nil {
		return nil, s.err
	}
	c := sampleAIConns()[0]
	return &c, nil
}

type stubAIUpdate struct {
	gotID string
	got   appintegrations.AIConnectionPatch
	err   error
}

func (s *stubAIUpdate) Execute(_ context.Context, id string, p appintegrations.AIConnectionPatch) (*integrationsDomain.AIConnection, error) {
	s.gotID, s.got = id, p
	if s.err != nil {
		return nil, s.err
	}
	c := sampleAIConns()[0]
	return &c, nil
}

type stubAIDelete struct {
	gotID string
	err   error
}

func (s *stubAIDelete) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

type stubAIReorder struct {
	got []string
	err error
}

func (s *stubAIReorder) Execute(_ context.Context, ids []string) ([]integrationsDomain.AIConnection, error) {
	s.got = ids
	return sampleAIConns(), s.err
}

type aiStubs struct {
	create  *stubAICreate
	update  *stubAIUpdate
	del     *stubAIDelete
	reorder *stubAIReorder
}

func newAIRouter(t *testing.T) (*gin.Engine, aiStubs) {
	t.Helper()
	s := aiStubs{create: &stubAICreate{}, update: &stubAIUpdate{}, del: &stubAIDelete{}, reorder: &stubAIReorder{}}
	h := newAIConnectionTestHandler(integrationshandler.AIConnectionUseCases{
		List: stubAIList{}, Create: s.create, Update: s.update, Delete: s.del, Reorder: s.reorder,
	})
	return newContractRouter(t, h), s
}

func TestListAIConnections_inOrderWithoutKeys(t *testing.T) {
	r, _ := newAIRouter(t)

	w := doRequest(r, http.MethodGet, "/api/v1/integrations/ai-connections", nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body genintegrations.AIConnectionListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 2)
	assert.Equal(t, "NVIDIA key 1", body.Data[0].Label)
	assert.True(t, body.Data[0].HasApiKey)
	assert.False(t, body.Data[1].HasApiKey)
	assert.False(t, body.Data[1].Enabled)
	assert.Equal(t, 1, body.Data[1].Position)
	assert.NotContains(t, w.Body.String(), "enc", "the stored key never leaves the server")
}

func TestCreateAIConnection_passesEverythingThrough(t *testing.T) {
	r, s := newAIRouter(t)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai-connections", map[string]any{
		"provider": "nvidia", "api_key": "nvapi-test", "model": "meta/llama-3.3-70b-instruct",
		"fallback_models": "meta/llama-3.1-8b-instruct", "label": "key 2", "enabled": false,
	})

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, integrationsDomain.AIProviderNVIDIA, s.create.got.Provider)
	assert.Equal(t, "nvapi-test", s.create.got.APIKey)
	assert.Equal(t, "key 2", s.create.got.Label)
	require.NotNil(t, s.create.got.Enabled)
	assert.False(t, *s.create.got.Enabled)
}

func TestCreateAIConnection_contractRejectsUnknownProviderAndExtraFields(t *testing.T) {
	r, s := newAIRouter(t)

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai-connections", map[string]any{"provider": "mistral", "api_key": "k"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	w = doRequest(r, http.MethodPost, "/api/v1/integrations/ai-connections", map[string]any{"provider": "openai", "api_key": "k", "position": 0})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, s.create.got.Provider)
}

func TestCreateAIConnection_domainValidationIs422(t *testing.T) {
	r, s := newAIRouter(t)
	s.create.err = integrationsDomain.ErrAPIKeyRequired

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai-connections", map[string]any{"provider": "openai"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "AI_API_KEY_REQUIRED")
}

func TestUpdateAIConnection_onlySentFields(t *testing.T) {
	r, s := newAIRouter(t)

	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/ai-connections/"+aiID2, map[string]any{"enabled": true, "label": "local"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, aiID2, s.update.gotID)
	require.NotNil(t, s.update.got.Enabled)
	assert.True(t, *s.update.got.Enabled)
	assert.Equal(t, "local", *s.update.got.Label)
	assert.Nil(t, s.update.got.APIKey, "no key sent keeps the stored one")
	assert.Nil(t, s.update.got.Provider)
}

func TestUpdateAIConnection_notFound(t *testing.T) {
	r, s := newAIRouter(t)
	s.update.err = integrationsDomain.ErrAIConnectionNotFound

	w := doRequest(r, http.MethodPatch, "/api/v1/integrations/ai-connections/"+aiID2, map[string]any{"label": "x"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAIConnection(t *testing.T) {
	r, s := newAIRouter(t)

	w := doRequest(r, http.MethodDelete, "/api/v1/integrations/ai-connections/"+aiID1, nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, aiID1, s.del.gotID)
}

func TestReorderAIConnections(t *testing.T) {
	r, s := newAIRouter(t)

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/ai-connections/order", map[string]any{"ids": []string{aiID2, aiID1}})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []string{aiID2, aiID1}, s.reorder.got)
}

func TestReorderAIConnections_incompleteOrderIs422(t *testing.T) {
	r, s := newAIRouter(t)
	s.reorder.err = integrationsDomain.ErrInvalidAIOrder

	w := doRequest(r, http.MethodPut, "/api/v1/integrations/ai-connections/order", map[string]any{"ids": []string{aiID2}})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "AI_INVALID_ORDER")
}
