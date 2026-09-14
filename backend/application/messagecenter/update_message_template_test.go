package messagecenter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

func TestUpdateMessageTemplate_success(t *testing.T) {
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{
		"template-1": {ID: "template-1", Name: "Old", Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "old", Enabled: true},
	}}
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 1), repo)

	tmpl, err := uc.Execute(context.Background(), appmc.UpdateMessageTemplateInput{
		ID: "template-1", Name: "New", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelSMS}, Body: "new body", Enabled: false,
	})

	require.NoError(t, err)
	require.Len(t, repo.updated, 1)
	assert.Equal(t, "New", tmpl.Name)
	assert.False(t, tmpl.Enabled)
}

func TestUpdateMessageTemplate_notFound(t *testing.T) {
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{}}
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), appmc.UpdateMessageTemplateInput{
		ID: "missing", Name: "New", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelSMS}, Body: "new body",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNotFound)
}

func TestUpdateMessageTemplate_rejectsInvalidInputBeforeLookup(t *testing.T) {
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{})

	_, err := uc.Execute(context.Background(), appmc.UpdateMessageTemplateInput{
		ID: "template-1", Name: "", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelSMS}, Body: "new body",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNameRequired)
}
