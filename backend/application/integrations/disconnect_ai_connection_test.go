package integrations_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
)

func TestDisconnectAIConnection_deletesConnection(t *testing.T) {
	deleteCalled := false
	repo := &mocks.AIConnectionRepo{
		DeleteFn: func(_ context.Context) error { deleteCalled = true; return nil },
	}
	uc := appintegrations.NewDisconnectAIConnectionUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.True(t, deleteCalled)
}

func TestDisconnectAIConnection_propagatesRepoError(t *testing.T) {
	repo := &mocks.AIConnectionRepo{
		DeleteFn: func(_ context.Context) error { return errors.New("db down") },
	}
	uc := appintegrations.NewDisconnectAIConnectionUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background())

	require.Error(t, err)
	assert.ErrorContains(t, err, "db down")
}
