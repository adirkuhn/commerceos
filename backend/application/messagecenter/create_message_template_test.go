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

func TestCreateMessageTemplate_success(t *testing.T) {
	repo := &stubMessageTemplateRepo{}
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	tmpl, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "Order confirmed", Category: mcDomain.CategoryPurchase,
		Channels: []messaging.Channel{messaging.ChannelEmail, messaging.ChannelWhatsApp},
		Subject:  "Your order is confirmed", Body: "Hi {{customer_name}}, order #{{order_number}} confirmed.",
	})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "Order confirmed", tmpl.Name)
	assert.True(t, tmpl.Enabled)
}

func TestCreateMessageTemplate_rejectsEmptyName(t *testing.T) {
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "x",
	})

	assert.ErrorIs(t, err, mcDomain.ErrTemplateNameRequired)
}

func TestCreateMessageTemplate_rejectsInvalidCategory(t *testing.T) {
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "X", Category: mcDomain.Category("bogus"), Channels: []messaging.Channel{messaging.ChannelEmail}, Body: "x",
	})

	assert.ErrorIs(t, err, mcDomain.ErrInvalidCategory)
}

func TestCreateMessageTemplate_rejectsNoChannels(t *testing.T) {
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "X", Category: mcDomain.CategoryPurchase, Body: "x",
	})

	assert.ErrorIs(t, err, mcDomain.ErrNoChannelsSelected)
}

func TestCreateMessageTemplate_rejectsInvalidChannel(t *testing.T) {
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "X", Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{"carrier-pigeon"}, Body: "x",
	})

	assert.ErrorIs(t, err, mcDomain.ErrInvalidChannel)
}

func TestCreateMessageTemplate_rejectsEmptyBody(t *testing.T) {
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 0), &stubMessageTemplateRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "X", Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{messaging.ChannelSMS},
	})
	assert.ErrorIs(t, err, mcDomain.ErrTemplateTextRequired)

	_, err = uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "X", Category: mcDomain.CategoryPurchase, Channels: []messaging.Channel{messaging.ChannelEmail}, TextBody: "only text",
	})
	assert.ErrorIs(t, err, mcDomain.ErrTemplateBodyRequired)
}

func TestCreateMessageTemplate_storesTextBodyAndDesign(t *testing.T) {
	repo := &stubMessageTemplateRepo{}
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	tmpl, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "Order", Category: mcDomain.CategoryOrderStatus, Channels: []messaging.Channel{messaging.ChannelEmail, messaging.ChannelWhatsApp},
		Body: "<p>Hi</p>", TextBody: "Hi", Design: []byte(`{"a":1}`),
	})

	require.NoError(t, err)
	assert.Equal(t, "Hi", tmpl.TextBody)
	assert.Equal(t, []byte(`{"a":1}`), repo.created[0].Design)
}

func TestCreateMessageTemplate_allThreeChannels_success(t *testing.T) {
	repo := &stubMessageTemplateRepo{}
	uc := appmc.NewCreateMessageTemplateUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	tmpl, err := uc.Execute(context.Background(), appmc.CreateMessageTemplateInput{
		Name: "Broadcast", Category: mcDomain.CategoryCustom,
		Channels: []messaging.Channel{messaging.ChannelEmail, messaging.ChannelSMS, messaging.ChannelWhatsApp},
		Body:     "Hi there",
	})

	require.NoError(t, err)
	assert.Len(t, tmpl.Channels, 3)
}
