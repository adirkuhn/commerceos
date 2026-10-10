package messagecenter_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
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
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

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
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.UpdateMessageTemplateInput{
		ID: "missing", Name: "New", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelSMS}, Body: "new body",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNotFound)
}

func TestUpdateMessageTemplate_rejectsInvalidInputBeforeLookup(t *testing.T) {
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.UpdateMessageTemplateInput{
		ID: "template-1", Name: "", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelSMS}, Body: "new body",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNameRequired)
}

func TestUpdateMessageTemplate_keepsDesignWhenOmittedAndReplacesWhenGiven(t *testing.T) {
	repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": {
		ID: "template-1", Name: "Order", Category: mcDomain.CategoryOrderStatus, Channels: []messaging.Channel{messaging.ChannelEmail},
		Body: "<p>old</p>", Design: []byte(`{"v":1}`),
	}}}
	uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 2), repo, &eventrecordtest.Outbox{})
	in := appmc.UpdateMessageTemplateInput{ID: "template-1", Name: "Order", Category: mcDomain.CategoryOrderStatus,
		Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "<p>old</p>", TextBody: "new", Enabled: true}

	tmpl, err := uc.Execute(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"v":1}`), tmpl.Design, "the simple form doesn't send a design; body unchanged")
	assert.Equal(t, "new", tmpl.TextBody)

	design := []byte(`{"v":2}`)
	in.Design = &design
	tmpl, err = uc.Execute(context.Background(), in)
	require.NoError(t, err)
	assert.Equal(t, design, tmpl.Design)
}

func TestUpdateMessageTemplate_bodyEditedOutsideTheEditorDropsTheStaleDesign(t *testing.T) {
	existing := mcDomain.MessageTemplate{ID: "tpl-1", Name: "X", Category: mcDomain.CategoryPurchase,
		Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "<p>old</p>", Design: []byte(`{"pages":[]}`)}
	in := appmc.UpdateMessageTemplateInput{ID: "tpl-1", Name: "X", Category: mcDomain.CategoryPurchase,
		Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true}

	for _, tc := range []struct {
		body       string
		wantDesign []byte
	}{
		{"<p>old</p>", []byte(`{"pages":[]}`)},
		{"<p>new</p>", nil},
	} {
		repo := &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"tpl-1": existing}}
		uc := appmc.NewUpdateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})
		in.Body = tc.body

		_, err := uc.Execute(context.Background(), in)

		require.NoError(t, err)
		assert.Equal(t, tc.wantDesign, repo.updated[0].Design, tc.body)
	}
}
