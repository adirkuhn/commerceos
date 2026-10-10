package notifications_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

var errBoom = errors.New("boom")

type stubPrefsRepo struct {
	saved     *notificationsDomain.Preferences
	getErr    error
	upsertErr error
	upserted  []notificationsDomain.Preferences
}

func (s *stubPrefsRepo) Get(context.Context) (*notificationsDomain.Preferences, error) {
	return s.saved, s.getErr
}

func (s *stubPrefsRepo) Upsert(_ context.Context, p *notificationsDomain.Preferences) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, *p)
	return nil
}

type outboxRow struct {
	subject, eventID string
	payload          []byte
}

type stubOutbox struct {
	err  error
	rows []outboxRow
}

func (s *stubOutbox) Enqueue(_ context.Context, subject, eventID string, payload []byte, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	s.rows = append(s.rows, outboxRow{subject, eventID, payload})
	return nil
}

func tctx() context.Context { return tenant.WithTenantID(context.Background(), "tenant-1") }

func prefsWith(cat notificationsDomain.Category, prefs notificationsDomain.ChannelPrefs, recipients ...string) *notificationsDomain.Preferences {
	p := notificationsDomain.DefaultPreferences("tenant-1")
	p.Categories[cat] = prefs
	p.EmailRecipients = recipients
	return p
}

// --- Get / Update preferences ---

func TestGetPreferences_defaultsWhenNoneSaved(t *testing.T) {
	uc := appnotifications.NewGetPreferencesUseCase(newMockGormDB(t), &stubPrefsRepo{})

	p, err := uc.Execute(tctx())

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", p.TenantID)
	assert.Equal(t, notificationsDomain.ChannelPrefs{InApp: true}, p.For(notificationsDomain.CategoryImports))
}

func TestGetPreferences_returnsSaved(t *testing.T) {
	saved := prefsWith(notificationsDomain.CategoryImports, notificationsDomain.ChannelPrefs{Email: true}, "ops@example.nl")
	uc := appnotifications.NewGetPreferencesUseCase(newMockGormDB(t), &stubPrefsRepo{saved: saved})

	p, err := uc.Execute(tctx())

	require.NoError(t, err)
	assert.Equal(t, []string{"ops@example.nl"}, p.EmailRecipients)
}

func TestGetPreferences_propagatesError(t *testing.T) {
	uc := appnotifications.NewGetPreferencesUseCase(newMockGormDB(t), &stubPrefsRepo{getErr: errBoom})
	_, err := uc.Execute(tctx())
	assert.True(t, errors.Is(err, errBoom))
}

func TestUpdatePreferences_savesAndRaisesEvent(t *testing.T) {
	repo, outbox := &stubPrefsRepo{}, &stubOutbox{}
	uc := appnotifications.NewUpdatePreferencesUseCase(newMockGormDB(t), repo, outbox)

	p, err := uc.Execute(tctx(), notificationsDomain.PreferencesInput{
		Categories:      map[notificationsDomain.Category]notificationsDomain.ChannelPrefs{notificationsDomain.CategoryImports: {Email: true}},
		EmailRecipients: []string{"ops@example.nl"},
	})

	require.NoError(t, err)
	assert.True(t, p.For(notificationsDomain.CategoryImports).Email)
	require.Len(t, repo.upserted, 1)
	assert.Equal(t, "tenant-1", repo.upserted[0].TenantID)
	require.Len(t, outbox.rows, 1)
	assert.Equal(t, "notifications.preferences.updated", outbox.rows[0].subject)
}

func TestUpdatePreferences_invalidInputWritesNothing(t *testing.T) {
	repo, outbox := &stubPrefsRepo{}, &stubOutbox{}
	uc := appnotifications.NewUpdatePreferencesUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(tctx(), notificationsDomain.PreferencesInput{EmailRecipients: []string{"nope"}})

	assert.True(t, errors.Is(err, notificationsDomain.ErrInvalidRecipient))
	assert.Empty(t, repo.upserted)
	assert.Empty(t, outbox.rows)
}

// --- Deliver ---

func deliverInput() appnotifications.CreateNotificationInput {
	return appnotifications.CreateNotificationInput{
		Type: notificationsDomain.TypeImportFailed, Title: "Import failed", Message: "Woo returned 500", Link: "/sync-log",
	}
}

func TestDeliver_defaultPreferencesStoreInAppOnly(t *testing.T) {
	notifRepo, outbox := &stubNotificationRepo{}, &stubOutbox{}
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), notifRepo, &stubPrefsRepo{}, outbox)

	n, err := uc.Execute(tctx(), deliverInput())

	require.NoError(t, err)
	require.NotNil(t, n)
	require.Len(t, notifRepo.created, 1)
	assert.Equal(t, "Import failed", notifRepo.created[0].Title)
	assert.Empty(t, outbox.rows)
}

func TestDeliver_emailOnEnqueuesEmailRequest(t *testing.T) {
	notifRepo, outbox := &stubNotificationRepo{}, &stubOutbox{}
	prefs := prefsWith(notificationsDomain.CategoryImports, notificationsDomain.ChannelPrefs{InApp: true, Email: true}, "ops@example.nl", "boss@example.nl")
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), notifRepo, &stubPrefsRepo{saved: prefs}, outbox)

	_, err := uc.Execute(tctx(), deliverInput())

	require.NoError(t, err)
	require.Len(t, notifRepo.created, 1)
	require.Len(t, outbox.rows, 1)
	assert.Equal(t, "notifications.email.requested", outbox.rows[0].subject)
	var ev struct {
		EventID    string   `json:"event_id"`
		TenantID   string   `json:"tenant_id"`
		Type       string   `json:"type"`
		Recipients []string `json:"recipients"`
		Subject    string   `json:"subject"`
		Body       string   `json:"body"`
		Link       string   `json:"link"`
	}
	require.NoError(t, json.Unmarshal(outbox.rows[0].payload, &ev))
	assert.Equal(t, outbox.rows[0].eventID, ev.EventID)
	assert.Equal(t, "tenant-1", ev.TenantID)
	assert.Equal(t, "import.failed", ev.Type)
	assert.Equal(t, []string{"ops@example.nl", "boss@example.nl"}, ev.Recipients)
	assert.Equal(t, "Import failed", ev.Subject)
	assert.Equal(t, "Woo returned 500", ev.Body)
	assert.Equal(t, "/sync-log", ev.Link)
}

func TestDeliver_inAppOffSkipsTheBell(t *testing.T) {
	notifRepo, outbox := &stubNotificationRepo{}, &stubOutbox{}
	prefs := prefsWith(notificationsDomain.CategoryImports, notificationsDomain.ChannelPrefs{InApp: false, Email: true}, "ops@example.nl")
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), notifRepo, &stubPrefsRepo{saved: prefs}, outbox)

	n, err := uc.Execute(tctx(), deliverInput())

	require.NoError(t, err)
	assert.Nil(t, n)
	assert.Empty(t, notifRepo.created)
	assert.Len(t, outbox.rows, 1)
}

func TestDeliver_propagatesErrors(t *testing.T) {
	prefs := prefsWith(notificationsDomain.CategoryImports, notificationsDomain.ChannelPrefs{InApp: true, Email: true}, "ops@example.nl")
	cases := map[string]*appnotifications.DeliverNotificationUseCase{
		"prefs":  appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), &stubNotificationRepo{}, &stubPrefsRepo{getErr: errBoom}, &stubOutbox{}),
		"create": appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), &stubNotificationRepo{createErr: errBoom}, &stubPrefsRepo{saved: prefs}, &stubOutbox{}),
		"outbox": appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), &stubNotificationRepo{}, &stubPrefsRepo{saved: prefs}, &stubOutbox{err: errBoom}),
	}
	for name, uc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := uc.Execute(tctx(), deliverInput())
			assert.True(t, errors.Is(err, errBoom))
		})
	}
}

func TestDeliver_messageCenterTypeSkipsGenericEmail(t *testing.T) {
	notifRepo, outbox := &stubNotificationRepo{}, &stubOutbox{}
	prefs := prefsWith(notificationsDomain.CategoryStock, notificationsDomain.ChannelPrefs{InApp: true, Email: true}, "ops@example.nl")
	uc := appnotifications.NewDeliverNotificationUseCase(newMockGormDB(t), notifRepo, &stubPrefsRepo{saved: prefs}, outbox)

	in := deliverInput()
	in.Type = notificationsDomain.TypeStockCloseToExpiry
	_, err := uc.Execute(tctx(), in)

	require.NoError(t, err)
	assert.Len(t, notifRepo.created, 1, "the bell still follows the in-app toggle")
	assert.Empty(t, outbox.rows, "the Message Center rule sends this email, not the generic notification email")
}

func TestCategoryEmailRecipients_returnsRecipientsWhenEmailOn(t *testing.T) {
	prefs := prefsWith(notificationsDomain.CategoryStock, notificationsDomain.ChannelPrefs{Email: true}, "ops@example.nl")
	uc := appnotifications.NewCategoryEmailRecipientsUseCase(newMockGormDB(t), &stubPrefsRepo{saved: prefs})

	got, err := uc.Execute(tctx(), notificationsDomain.CategoryStock)

	require.NoError(t, err)
	assert.Equal(t, []string{"ops@example.nl"}, got)
}

func TestCategoryEmailRecipients_emptyWhenEmailOff(t *testing.T) {
	prefs := prefsWith(notificationsDomain.CategoryStock, notificationsDomain.ChannelPrefs{InApp: true}, "ops@example.nl")
	uc := appnotifications.NewCategoryEmailRecipientsUseCase(newMockGormDB(t), &stubPrefsRepo{saved: prefs})

	got, err := uc.Execute(tctx(), notificationsDomain.CategoryStock)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCategoryEmailRecipients_emptyWhenNeverSaved(t *testing.T) {
	uc := appnotifications.NewCategoryEmailRecipientsUseCase(newMockGormDB(t), &stubPrefsRepo{})

	got, err := uc.Execute(tctx(), notificationsDomain.CategoryStock)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCategoryEmailRecipients_propagatesError(t *testing.T) {
	uc := appnotifications.NewCategoryEmailRecipientsUseCase(newMockGormDB(t), &stubPrefsRepo{getErr: errBoom})

	_, err := uc.Execute(tctx(), notificationsDomain.CategoryStock)

	assert.ErrorIs(t, err, errBoom)
}
