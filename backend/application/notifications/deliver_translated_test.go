package notifications_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

func TestDeliver_storesParamsAndEnglishFallbackText(t *testing.T) {
	notifRepo, outbox := &stubNotificationRepo{}, &stubOutbox{}
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), notifRepo, &stubPrefsRepo{}, outbox)

	_, err := uc.Execute(tctx(), appnotifications.CreateNotificationInput{
		Type: notificationsDomain.TypeBolNotForSale, Params: map[string]any{"product": "Guaraná", "reason": "EAN unknown"}, Link: "/products/p-1",
	})

	require.NoError(t, err)
	require.Len(t, notifRepo.created, 1)
	n := notifRepo.created[0]
	assert.Equal(t, map[string]any{"product": "Guaraná", "reason": "EAN unknown"}, n.Params)
	assert.Equal(t, `Bol: "Guaraná" is not for sale`, n.Title, "English text stays stored for old clients")
	assert.Equal(t, "EAN unknown", n.Message)
}

func TestDeliver_untranslatedTypeKeepsItsGivenText(t *testing.T) {
	notifRepo := &stubNotificationRepo{}
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), notifRepo, &stubPrefsRepo{}, &stubOutbox{})

	_, err := uc.Execute(tctx(), appnotifications.CreateNotificationInput{Type: "something.new", Title: "Hello", Message: "World"})

	require.NoError(t, err)
	assert.Equal(t, "Hello", notifRepo.created[0].Title)
	assert.Equal(t, map[string]any{}, notifRepo.created[0].Params)
}

func TestDeliver_emailRequestCarriesParams(t *testing.T) {
	outbox := &stubOutbox{}
	prefs := prefsWith(notificationsDomain.CategoryImports, notificationsDomain.ChannelPrefs{InApp: true, Email: true}, "ops@example.nl")
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), &stubNotificationRepo{}, &stubPrefsRepo{saved: prefs}, outbox)

	_, err := uc.Execute(tctx(), appnotifications.CreateNotificationInput{Type: notificationsDomain.TypeImportFailed, Params: map[string]any{"error": "Woo returned 500"}})

	require.NoError(t, err)
	require.Len(t, outbox.rows, 1)
	var ev struct {
		Type    string         `json:"type"`
		Params  map[string]any `json:"params"`
		Subject string         `json:"subject"`
	}
	require.NoError(t, json.Unmarshal(outbox.rows[0].payload, &ev))
	assert.Equal(t, "import.failed", ev.Type)
	assert.Equal(t, map[string]any{"error": "Woo returned 500"}, ev.Params)
	assert.Equal(t, "Import failed", ev.Subject, "English fallback for consumers that predate params")
}
