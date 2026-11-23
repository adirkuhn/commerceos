package integrations_test

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	integrationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/integrations"
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

func TestConnectAIProviderHandler_contractAcceptsNVIDIA(t *testing.T) {
	connect := &stubConnectAIProvider{}
	r := newContractRouter(t, newAIConnectionTestHandler(&stubGetAIConnection{}, connect, &stubDisconnectAIConnection{}))

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai", map[string]any{
		"provider": "nvidia", "api_key": "nvapi-test", "model": "meta/llama-3.3-70b-instruct",
		"fallback_models": "meta/llama-3.1-8b-instruct",
	})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, integrationsDomain.AIProviderNVIDIA, connect.gotInput.Provider)
	assert.Equal(t, "nvapi-test", connect.gotInput.APIKey)
	assert.Equal(t, "meta/llama-3.1-8b-instruct", connect.gotInput.FallbackModels)
}

func TestConnectAIProviderHandler_contractRejectsUnknownProvider(t *testing.T) {
	connect := &stubConnectAIProvider{}
	r := newContractRouter(t, newAIConnectionTestHandler(&stubGetAIConnection{}, connect, &stubDisconnectAIConnection{}))

	w := doRequest(r, http.MethodPost, "/api/v1/integrations/ai", map[string]any{"provider": "mistral", "api_key": "k"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, connect.gotInput.Provider, "the use case is never reached")
}
