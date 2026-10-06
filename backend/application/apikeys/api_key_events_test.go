package apikeys_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
)

func TestCreateAPIKey_RecordsCreatedWithoutTheKey(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	channel := "ch-1"

	out, err := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{}, outbox).Execute(context.Background(), "Woo", "shop", &channel)

	require.NoError(t, err)
	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "apikeys.api_key.created", e.Subject)
	assert.Equal(t, "key-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Woo", "sales_channel_id": "ch-1"}, e.Data())
	raw, err := json.Marshal(e.Payload)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), out.PlaintextKey)
}

func TestRevokeAPIKey_RecordsRevoked(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	require.NoError(t, appapikeys.NewRevokeAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{}, outbox).Execute(context.Background(), "key-1"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "apikeys.api_key.revoked", e.Subject)
	assert.Equal(t, "key-1", e.Payload["entity_id"])
}

func TestAPIKeyUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	_, err := appapikeys.NewCreateAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{}, &eventrecordtest.Outbox{Err: boom}).Execute(context.Background(), "Woo", "", nil)
	assert.ErrorIs(t, err, boom)
	assert.ErrorIs(t, appapikeys.NewRevokeAPIKeyUseCase(newMockGormDB(t), &stubAPIKeyRepo{}, &eventrecordtest.Outbox{Err: boom}).Execute(context.Background(), "key-1"), boom)
}
