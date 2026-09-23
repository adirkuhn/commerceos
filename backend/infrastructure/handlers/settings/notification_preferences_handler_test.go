package settings_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
)

type stubGetNotificationPrefs struct {
	out *notificationsDomain.Preferences
	err error
}

func (s *stubGetNotificationPrefs) Execute(context.Context) (*notificationsDomain.Preferences, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return notificationsDomain.DefaultPreferences("tenant-1"), nil
}

type stubUpdateNotificationPrefs struct {
	err error
	got notificationsDomain.PreferencesInput
}

func (s *stubUpdateNotificationPrefs) Execute(_ context.Context, in notificationsDomain.PreferencesInput) (*notificationsDomain.Preferences, error) {
	s.got = in
	if s.err != nil {
		return nil, s.err
	}
	p, err := notificationsDomain.NewPreferences("tenant-1", in, time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC))
	return p, err
}

type prefsEnvelope struct {
	Data struct {
		Categories []struct {
			Category string `json:"category"`
			InApp    bool   `json:"in_app"`
			Email    bool   `json:"email"`
		} `json:"categories"`
		EmailRecipients []string `json:"email_recipients"`
		UpdatedAt       *string  `json:"updated_at"`
	} `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestGetNotificationPreferences_listsEveryCategoryInOrder(t *testing.T) {
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}))

	w := doRequest(r, http.MethodGet, "/api/v1/settings/notification-preferences", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var body prefsEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Categories, 4)
	assert.Equal(t, "imports", body.Data.Categories[0].Category)
	assert.True(t, body.Data.Categories[0].InApp)
	assert.False(t, body.Data.Categories[0].Email)
	assert.Equal(t, []string{}, body.Data.EmailRecipients)
	assert.Nil(t, body.Data.UpdatedAt, "never saved")
}

func TestUpdateNotificationPreferences_passesInputThrough(t *testing.T) {
	update := &stubUpdateNotificationPrefs{}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, update))

	w := doRequest(r, http.MethodPut, "/api/v1/settings/notification-preferences", map[string]any{
		"categories":       []map[string]any{{"category": "imports", "in_app": false, "email": true}},
		"email_recipients": []string{"ops@example.nl"},
	})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, notificationsDomain.ChannelPrefs{InApp: false, Email: true}, update.got.Categories[notificationsDomain.CategoryImports])
	assert.Equal(t, []string{"ops@example.nl"}, update.got.EmailRecipients)
	var body prefsEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.True(t, body.Data.Categories[0].Email)
	require.NotNil(t, body.Data.UpdatedAt)
}

func TestUpdateNotificationPreferences_mapsValidationErrors(t *testing.T) {
	cases := map[error]string{
		notificationsDomain.ErrInvalidRecipient:        "INVALID_RECIPIENT",
		notificationsDomain.ErrTooManyRecipients:       "TOO_MANY_RECIPIENTS",
		notificationsDomain.ErrEmailRecipientsRequired: "EMAIL_RECIPIENTS_REQUIRED",
		notificationsDomain.ErrUnknownCategory:         "UNKNOWN_CATEGORY",
	}
	for domainErr, code := range cases {
		t.Run(code, func(t *testing.T) {
			update := &stubUpdateNotificationPrefs{err: fmt.Errorf("wrap: %w", domainErr)}
			r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, update))

			w := doRequest(r, http.MethodPut, "/api/v1/settings/notification-preferences", map[string]any{
				"categories": []map[string]any{}, "email_recipients": []string{},
			})

			require.Equal(t, http.StatusUnprocessableEntity, w.Code)
			var body prefsEnvelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, code, body.Error.Code)
		})
	}
}
