package messagecenter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubSentEmailRepo struct {
	created   []mcDomain.SentEmail
	list      []mcDomain.SentEmail
	total     int
	gotFilter mcDomain.SentEmailFilter
	byID      map[string]mcDomain.SentEmail
	deleted   []string
	err       error
}

func (s *stubSentEmailRepo) Create(_ context.Context, e *mcDomain.SentEmail) error {
	if s.err != nil {
		return s.err
	}
	e.ID = "sent-1"
	s.created = append(s.created, *e)
	return nil
}
func (s *stubSentEmailRepo) List(_ context.Context, f mcDomain.SentEmailFilter) ([]mcDomain.SentEmail, int, error) {
	s.gotFilter = f
	return s.list, s.total, s.err
}
func (s *stubSentEmailRepo) FindByID(_ context.Context, id string) (*mcDomain.SentEmail, error) {
	e, ok := s.byID[id]
	if !ok {
		return nil, mcDomain.ErrSentEmailNotFound
	}
	return &e, nil
}
func (s *stubSentEmailRepo) Delete(_ context.Context, ids []string) ([]string, error) {
	var out []string
	for _, id := range ids {
		if _, ok := s.byID[id]; ok {
			out = append(out, id)
		}
	}
	s.deleted = out
	return out, s.err
}

func tenantCtx() context.Context { return tenant.AsTenant(context.Background(), "tenant-1") }

func TestRecordSentEmail_storesItForTheTenantAndRaisesAnEvent(t *testing.T) {
	repo, outbox := &stubSentEmailRepo{}, &eventrecordtest.Outbox{}
	uc := appmc.NewRecordSentEmailUseCase(newMockGormDB(t, 1), repo, outbox)

	err := uc.Execute(tenantCtx(), mcDomain.SentEmail{Source: mcDomain.SentEmailSourceNotification, Recipient: "a@b.c", Subject: "S", Status: mcDomain.SentEmailSent})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "tenant-1", repo.created[0].TenantID)
	assert.False(t, repo.created[0].CreatedAt.IsZero())
	require.Len(t, outbox.Events, 1)
	assert.NotContains(t, outbox.Events[0].Payload, "recipient", "addresses stay out of events")
}

func TestRecordSentEmail_withoutATenantRecordsNothing(t *testing.T) {
	repo := &stubSentEmailRepo{}
	uc := appmc.NewRecordSentEmailUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	require.NoError(t, uc.Execute(context.Background(), mcDomain.SentEmail{Status: mcDomain.SentEmailSent}))
	assert.Empty(t, repo.created)
}

type stubEmailClient struct {
	err error
	got []messaging.Message
}

func (c *stubEmailClient) Send(_ context.Context, m messaging.Message) error {
	c.got = append(c.got, m)
	return c.err
}

type recordedEmails struct {
	got []mcDomain.SentEmail
	err error
}

func (r *recordedEmails) Execute(_ context.Context, e mcDomain.SentEmail) error {
	r.got = append(r.got, e)
	return r.err
}

func TestRecordingEmailClient_recordsEverySend(t *testing.T) {
	inner, rec := &stubEmailClient{}, &recordedEmails{}
	client := appmc.NewRecordingEmailClient(inner, rec, mcDomain.SentEmailSourceSignInLink)
	msg := messaging.Message{Channel: messaging.ChannelEmail, To: "a@b.c", Subject: "Sign in", HTMLBody: "<a>link</a>"}

	require.NoError(t, client.Send(tenantCtx(), msg))

	assert.Equal(t, []messaging.Message{msg}, inner.got)
	require.Len(t, rec.got, 1)
	assert.Equal(t, mcDomain.SentEmail{Source: mcDomain.SentEmailSourceSignInLink, Recipient: "a@b.c", Subject: "Sign in",
		HTMLBody: "<a>link</a>", Status: mcDomain.SentEmailSent}, rec.got[0])
}

func TestRecordingEmailClient_recordsAFailureAndStillReturnsIt(t *testing.T) {
	boom := errors.New("550 mailbox unavailable")
	rec := &recordedEmails{}
	client := appmc.NewRecordingEmailClient(&stubEmailClient{err: boom}, rec, mcDomain.SentEmailSourceNotification)

	err := client.Send(tenantCtx(), messaging.Message{To: "a@b.c", HTMLBody: "<p/>"})

	assert.ErrorIs(t, err, boom)
	require.Len(t, rec.got, 1)
	assert.Equal(t, mcDomain.SentEmailFailed, rec.got[0].Status)
	assert.Equal(t, "550 mailbox unavailable", rec.got[0].Reason)
}

func TestRecordingEmailClient_aHistoryErrorDoesNotFailTheSend(t *testing.T) {
	client := appmc.NewRecordingEmailClient(&stubEmailClient{}, &recordedEmails{err: errors.New("db down")}, mcDomain.SentEmailSourceNewsletter)

	assert.NoError(t, client.Send(tenantCtx(), messaging.Message{To: "a@b.c"}), "the email went out")
}

func TestRecordingEmailClient_plainTextBodyIsKept(t *testing.T) {
	rec := &recordedEmails{}
	client := appmc.NewRecordingEmailClient(&stubEmailClient{}, rec, mcDomain.SentEmailSourceNotification)

	require.NoError(t, client.Send(tenantCtx(), messaging.Message{To: "a@b.c", Body: "plain <text>"}))

	assert.Equal(t, "<pre>plain &lt;text&gt;</pre>", rec.got[0].HTMLBody)
}

func TestListSentEmails(t *testing.T) {
	repo := &stubSentEmailRepo{list: []mcDomain.SentEmail{{ID: "a"}}, total: 7}
	uc := appmc.NewListSentEmailsUseCase(newMockGormDB(t, 1), repo)

	page, err := uc.Execute(tenantCtx(), mcDomain.SentEmailFilter{Status: mcDomain.SentEmailFailed, PerPage: 500})

	require.NoError(t, err)
	assert.Equal(t, 7, page.Total)
	assert.Equal(t, 1, page.Page)
	assert.Equal(t, 200, page.PerPage)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, mcDomain.SentEmailFailed, repo.gotFilter.Status)
}

func TestListSentEmails_rejectsUnknownFilters(t *testing.T) {
	uc := appmc.NewListSentEmailsUseCase(newMockGormDB(t, 0), &stubSentEmailRepo{})

	_, err := uc.Execute(tenantCtx(), mcDomain.SentEmailFilter{Status: "bounced"})
	assert.ErrorIs(t, err, mcDomain.ErrSentEmailFilter)
	_, err = uc.Execute(tenantCtx(), mcDomain.SentEmailFilter{Source: "fax"})
	assert.ErrorIs(t, err, mcDomain.ErrSentEmailFilter)
}

func TestGetSentEmail(t *testing.T) {
	repo := &stubSentEmailRepo{byID: map[string]mcDomain.SentEmail{"a": {ID: "a", HTMLBody: "<p>x</p>"}}}
	uc := appmc.NewGetSentEmailUseCase(newMockGormDB(t, 2), repo)

	e, err := uc.Execute(tenantCtx(), "a")
	require.NoError(t, err)
	assert.Equal(t, "<p>x</p>", e.HTMLBody)

	_, err = uc.Execute(tenantCtx(), "missing")
	assert.ErrorIs(t, err, mcDomain.ErrSentEmailNotFound)
}

func TestDeleteSentEmails(t *testing.T) {
	repo := &stubSentEmailRepo{byID: map[string]mcDomain.SentEmail{"a": {ID: "a"}, "b": {ID: "b"}}}
	outbox := &eventrecordtest.Outbox{}
	uc := appmc.NewDeleteSentEmailsUseCase(newMockGormDB(t, 1), repo, outbox)

	n, err := uc.Execute(tenantCtx(), []string{"a", "b", "gone"})

	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Len(t, outbox.Events, 2, "one deleted event per removed entry")
}

func TestDeleteSentEmails_errors(t *testing.T) {
	uc := appmc.NewDeleteSentEmailsUseCase(newMockGormDB(t, 1), &stubSentEmailRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(tenantCtx(), nil)
	assert.ErrorIs(t, err, mcDomain.ErrNoSentEmailsGiven)

	_, err = uc.Execute(tenantCtx(), []string{"gone"})
	assert.ErrorIs(t, err, mcDomain.ErrSentEmailNotFound, "deleting a single missing entry is a 404")
}


type stubFactory struct{ client messaging.Client }

func (f stubFactory) NewClient(_, _, _, _, _ string) messaging.Client { return f.client }

func TestRecordingEmailClientFactory_wrapsEveryClientItMakes(t *testing.T) {
	inner, rec := &stubEmailClient{}, &recordedEmails{}
	factory := appmc.NewRecordingEmailClientFactory(stubFactory{client: inner}, rec, mcDomain.SentEmailSourceNotification)

	client := factory.NewClient("smtp.example.com", "587", "u", "p", "from@example.com")
	require.NoError(t, client.Send(tenantCtx(), messaging.Message{To: "a@b.c", HTMLBody: "<p/>"}))

	assert.Len(t, inner.got, 1)
	require.Len(t, rec.got, 1)
	assert.Equal(t, mcDomain.SentEmailSourceNotification, rec.got[0].Source)
}
