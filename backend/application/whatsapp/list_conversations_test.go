package whatsapp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
)

func TestListWhatsAppConversations_returnsRepoResult(t *testing.T) {
	repo := &fakeMessageRepo{}
	require.NoError(t, repo.Create(context.Background(), &whatsappDomain.Message{
		PhoneE164: "31612345678", Direction: whatsappDomain.MessageDirectionInbound, Body: "hi",
	}))
	uc := appwhatsapp.NewListWhatsAppConversationsUseCase(newMockGormDB(t, 1), repo)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "31612345678", out[0].PhoneE164)
}

func TestListWhatsAppMessagesByPhone_returnsRepoResult(t *testing.T) {
	repo := &fakeMessageRepo{}
	require.NoError(t, repo.Create(context.Background(), &whatsappDomain.Message{
		PhoneE164: "31612345678", Direction: whatsappDomain.MessageDirectionInbound, Body: "hi",
	}))
	require.NoError(t, repo.Create(context.Background(), &whatsappDomain.Message{
		PhoneE164: "31699999999", Direction: whatsappDomain.MessageDirectionInbound, Body: "oi",
	}))
	uc := appwhatsapp.NewListWhatsAppMessagesByPhoneUseCase(newMockGormDB(t, 1), repo)

	out, err := uc.Execute(context.Background(), "31612345678")

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "hi", out[0].Body)
}
