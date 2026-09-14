package messagecenter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestDeleteMessageTemplate_success(t *testing.T) {
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": {ID: "template-1"}}}
	uc := appmc.NewDeleteMessageTemplateUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "template-1")

	require.NoError(t, err)
	assert.Equal(t, "template-1", repo.deletedID)
}

func TestDeleteMessageTemplate_notFound(t *testing.T) {
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{}}
	uc := appmc.NewDeleteMessageTemplateUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNotFound)
}
