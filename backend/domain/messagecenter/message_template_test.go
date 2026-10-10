package messagecenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

func TestCategory_Valid(t *testing.T) {
	valid := []messagecenter.Category{
		messagecenter.CategoryPurchase, messagecenter.CategoryOrderStatus,
		messagecenter.CategoryStockInfo, messagecenter.CategoryTracking, messagecenter.CategoryCustom,
	}
	for _, c := range valid {
		assert.True(t, c.Valid(), "expected %q to be valid", c)
	}
	assert.False(t, messagecenter.Category("bogus").Valid())
	assert.False(t, messagecenter.Category("").Valid())
}

func TestMessageTemplate_BodyFor(t *testing.T) {
	tmpl := messagecenter.MessageTemplate{Body: "<p>Hi</p>", TextBody: "Hi"}
	assert.Equal(t, "<p>Hi</p>", tmpl.BodyFor(messaging.ChannelEmail))
	assert.Equal(t, "Hi", tmpl.BodyFor(messaging.ChannelWhatsApp))
	assert.Equal(t, "Hi", tmpl.BodyFor(messaging.ChannelSMS))

	legacy := messagecenter.MessageTemplate{Body: "Hi {{customer_name}}"}
	assert.Equal(t, "Hi {{customer_name}}", legacy.BodyFor(messaging.ChannelWhatsApp), "templates from before text bodies keep using Body")
}

func TestValidateContent(t *testing.T) {
	email := []messaging.Channel{messaging.ChannelEmail}
	both := []messaging.Channel{messaging.ChannelEmail, messaging.ChannelWhatsApp}
	whatsapp := []messaging.Channel{messaging.ChannelWhatsApp}

	assert.NoError(t, messagecenter.ValidateContent(email, "<p>x</p>", ""))
	assert.ErrorIs(t, messagecenter.ValidateContent(email, "", "text"), messagecenter.ErrTemplateBodyRequired)
	assert.NoError(t, messagecenter.ValidateContent(whatsapp, "", "Hi"))
	assert.NoError(t, messagecenter.ValidateContent(whatsapp, "Hi", ""), "legacy body serves WhatsApp")
	assert.ErrorIs(t, messagecenter.ValidateContent(whatsapp, "", ""), messagecenter.ErrTemplateTextRequired)
	assert.NoError(t, messagecenter.ValidateContent(both, "<p>x</p>", ""))
}
