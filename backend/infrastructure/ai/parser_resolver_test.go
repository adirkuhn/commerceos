package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/ai"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

func TestParserResolver_returnsNilWhenNotConfigured(t *testing.T) {
	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) { return nil, nil }}
	resolver := ai.NewParserResolver(newMockGormDB(t), repo, "test-secret")

	parser, err := resolver.Resolve(context.Background())

	require.NoError(t, err)
	assert.Nil(t, parser)
}

func TestParserResolver_buildsCorrectAdapterPerProvider(t *testing.T) {
	cases := []struct {
		provider integrationsDomain.AIProvider
		baseURL  string
		wantType any
	}{
		{integrationsDomain.AIProviderOpenAI, "", &ai.OpenAIDocumentParser{}},
		{integrationsDomain.AIProviderAnthropic, "", &ai.AnthropicDocumentParser{}},
		{integrationsDomain.AIProviderGemini, "", &ai.GeminiDocumentParser{}},
		{integrationsDomain.AIProviderOllama, "http://localhost:11434", &ai.OllamaDocumentParser{}},
	}
	for _, tc := range cases {
		t.Run(string(tc.provider), func(t *testing.T) {
			repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
				return &integrationsDomain.AIConnection{Provider: tc.provider, BaseURL: tc.baseURL}, nil
			}}
			resolver := ai.NewParserResolver(newMockGormDB(t), repo, "test-secret")

			parser, err := resolver.Resolve(context.Background())

			require.NoError(t, err)
			require.NotNil(t, parser)
			assert.IsType(t, tc.wantType, parser)
		})
	}
}

func TestParserResolver_decryptsAPIKeyForTheAdapterToUse(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{"type": "message", "content": []map[string]any{{"type": "output_text", "text": `{"line_items":[],"supplier_name":""}`}}},
			},
		})
	}))
	defer server.Close()

	encryptedKey, err := crypto.Encrypt("test-secret", "sk-real-key")
	require.NoError(t, err)

	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{Provider: integrationsDomain.AIProviderOpenAI, APIKeyEncrypted: encryptedKey, BaseURL: server.URL}, nil
	}}
	resolver := ai.NewParserResolver(newMockGormDB(t), repo, "test-secret")

	parser, err := resolver.Resolve(context.Background())
	require.NoError(t, err)

	_, err = parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{FileName: "x.pdf", Data: []byte("%PDF")})
	require.NoError(t, err)
	assert.Equal(t, "Bearer sk-real-key", capturedAuth)
}

func TestParserResolver_propagatesRepoError(t *testing.T) {
	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return nil, errors.New("db down")
	}}
	resolver := ai.NewParserResolver(newMockGormDB(t), repo, "test-secret")

	_, err := resolver.Resolve(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}

func TestParserResolver_propagatesUnsupportedProvider(t *testing.T) {
	repo := &mocks.AIConnectionRepo{GetFn: func(_ context.Context) (*integrationsDomain.AIConnection, error) {
		return &integrationsDomain.AIConnection{Provider: integrationsDomain.AIProvider("deepseek")}, nil
	}}
	resolver := ai.NewParserResolver(newMockGormDB(t), repo, "test-secret")

	_, err := resolver.Resolve(context.Background())

	assert.ErrorIs(t, err, integrationsDomain.ErrProviderNotSupported)
}
