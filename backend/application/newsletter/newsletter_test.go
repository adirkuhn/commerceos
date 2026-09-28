package newsletter_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appnewsletter "github.com/adirkuhn/commerceos/application/newsletter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/domain/newsletter"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func mockDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(false)
	for i := 0; i < 20; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
		mock.ExpectRollback()
	}
	return db
}

type fakeRepo struct {
	byID map[string]newsletter.Subscriber
}

func (f *fakeRepo) Create(_ context.Context, s *newsletter.Subscriber) error {
	s.ID = fmt.Sprintf("s%d", len(f.byID)+1)
	f.byID[s.ID] = *s
	return nil
}
func (f *fakeRepo) find(match func(newsletter.Subscriber) bool) (*newsletter.Subscriber, error) {
	for _, s := range f.byID {
		if match(s) {
			return &s, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) FindByEmail(_ context.Context, e string) (*newsletter.Subscriber, error) {
	return f.find(func(s newsletter.Subscriber) bool { return s.Email == e })
}
func (f *fakeRepo) FindByConfirmTokenHash(_ context.Context, h string) (*newsletter.Subscriber, error) {
	return f.find(func(s newsletter.Subscriber) bool { return s.ConfirmTokenHash == h && h != "" })
}
func (f *fakeRepo) FindByUnsubscribeToken(_ context.Context, tok string) (*newsletter.Subscriber, error) {
	return f.find(func(s newsletter.Subscriber) bool { return s.UnsubscribeToken == tok })
}
func (f *fakeRepo) Update(_ context.Context, s *newsletter.Subscriber) error {
	f.byID[s.ID] = *s
	return nil
}
func (f *fakeRepo) List(_ context.Context, fl newsletter.ListFilter) ([]newsletter.Subscriber, int64, error) {
	var out []newsletter.Subscriber
	for _, s := range f.byID {
		if fl.Status == nil || s.Status == *fl.Status {
			out = append(out, s)
		}
	}
	total := int64(len(out))
	if fl.Offset >= len(out) {
		return nil, total, nil
	}
	out = out[fl.Offset:]
	if fl.Limit < len(out) {
		out = out[:fl.Limit]
	}
	return out, total, nil
}
func (f *fakeRepo) Delete(_ context.Context, id string) error {
	if _, ok := f.byID[id]; !ok {
		return newsletter.ErrSubscriberNotFound
	}
	delete(f.byID, id)
	return nil
}

type fakeMailer struct{ sent []messaging.Message }

func (f *fakeMailer) Send(_ context.Context, m messaging.Message) error {
	f.sent = append(f.sent, m)
	return nil
}

type fakeOutbox struct{ subjects []string }

func (f *fakeOutbox) Enqueue(_ context.Context, s, _ string, _ []byte, _ time.Time) error {
	f.subjects = append(f.subjects, s)
	return nil
}

type fakeStores struct{}

func (fakeStores) StoreName(_ context.Context, _ string) (string, error) { return "Ann's <Shop>", nil }

type world struct {
	deps   appnewsletter.Deps
	repo   *fakeRepo
	mail   *fakeMailer
	outbox *fakeOutbox
	clock  time.Time
}

func newWorld(t *testing.T) *world {
	w := &world{repo: &fakeRepo{byID: map[string]newsletter.Subscriber{}}, mail: &fakeMailer{}, outbox: &fakeOutbox{}, clock: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
	n := 0
	w.deps = appnewsletter.Deps{
		DB: mockDB(t), Repo: w.repo, Mailer: w.mail, Outbox: w.outbox, Stores: fakeStores{},
		Now:      func() time.Time { return w.clock },
		NewToken: func() (string, error) { n++; return fmt.Sprintf("tok%d", n), nil },
		ConfirmURL: func(channelID, token string) string {
			return "https://shop.test/store/" + channelID + "/newsletter/confirm?token=" + token
		},
		UnsubscribeURL: func(channelID, token string) string {
			return "https://shop.test/store/" + channelID + "/newsletter/unsubscribe?token=" + token
		},
	}
	return w
}

var ctx = tenant.WithTenantID(context.Background(), "tenant-1")

func subscribe(w *world, email string) error {
	return appnewsletter.NewSubscribeUseCase(w.deps).Execute(ctx, appnewsletter.SubscribeInput{SalesChannelID: "chan-1", Email: email, Consent: true})
}

func TestSubscribe_sendsAConfirmationLink(t *testing.T) {
	w := newWorld(t)

	require.NoError(t, subscribe(w, " Ann@Example.com "))

	require.Len(t, w.mail.sent, 1)
	m := w.mail.sent[0]
	assert.Equal(t, "ann@example.com", m.To)
	assert.Contains(t, m.HTMLBody, "https://shop.test/store/chan-1/newsletter/confirm?token=tok1")
	assert.Contains(t, m.HTMLBody, "Ann&#39;s &lt;Shop&gt;", "store name is HTML-escaped")
	assert.NotContains(t, m.HTMLBody, "<Shop>")
	s, _ := w.repo.FindByEmail(ctx, "ann@example.com")
	assert.Equal(t, newsletter.StatusPending, s.Status)
	assert.Equal(t, newsletter.HashToken("tok1"), s.ConfirmTokenHash)
}

func TestSubscribe_validation(t *testing.T) {
	w := newWorld(t)
	assert.ErrorIs(t, subscribe(w, "nope"), newsletter.ErrInvalidEmail)
	err := appnewsletter.NewSubscribeUseCase(w.deps).Execute(ctx, appnewsletter.SubscribeInput{SalesChannelID: "chan-1", Email: "a@x.io"})
	assert.ErrorIs(t, err, newsletter.ErrConsentRequired)
	assert.Empty(t, w.mail.sent)
}

func TestSubscribe_repeatSignupsAreThrottledAndLookTheSame(t *testing.T) {
	w := newWorld(t)
	require.NoError(t, subscribe(w, "a@x.io"))
	require.NoError(t, subscribe(w, "a@x.io"))
	assert.Len(t, w.mail.sent, 1, "no second email within the throttle window")

	w.clock = w.clock.Add(newsletter.ResendAfter)
	require.NoError(t, subscribe(w, "a@x.io"))
	assert.Len(t, w.mail.sent, 2)
	assert.Len(t, w.repo.byID, 1)
}

func TestConfirmAndUnsubscribe(t *testing.T) {
	w := newWorld(t)
	require.NoError(t, subscribe(w, "a@x.io"))

	require.NoError(t, appnewsletter.NewConfirmUseCase(w.deps).Execute(ctx, "tok1"))
	s, _ := w.repo.FindByEmail(ctx, "a@x.io")
	assert.Equal(t, newsletter.StatusConfirmed, s.Status)
	assert.Equal(t, []string{"newsletter.subscriber.confirmed"}, w.outbox.subjects)
	assert.ErrorIs(t, appnewsletter.NewConfirmUseCase(w.deps).Execute(ctx, "tok1"), newsletter.ErrInvalidToken, "links work once")
	assert.ErrorIs(t, appnewsletter.NewConfirmUseCase(w.deps).Execute(ctx, "bogus"), newsletter.ErrInvalidToken)

	require.NoError(t, subscribe(w, "a@x.io"))
	assert.Len(t, w.mail.sent, 1, "already confirmed: nothing sent")

	require.NoError(t, appnewsletter.NewUnsubscribeUseCase(w.deps).Execute(ctx, s.UnsubscribeToken))
	s, _ = w.repo.FindByEmail(ctx, "a@x.io")
	assert.Equal(t, newsletter.StatusUnsubscribed, s.Status)
	assert.ErrorIs(t, appnewsletter.NewUnsubscribeUseCase(w.deps).Execute(ctx, "bogus"), newsletter.ErrInvalidToken)
}

func TestConfirmationEmailCarriesTheUnsubscribeLink(t *testing.T) {
	w := newWorld(t)
	require.NoError(t, subscribe(w, "a@x.io"))
	s, _ := w.repo.FindByEmail(ctx, "a@x.io")
	assert.Contains(t, w.mail.sent[0].HTMLBody, "newsletter/unsubscribe?token="+s.UnsubscribeToken)
}

func TestAdmin_listExportAndErase(t *testing.T) {
	w := newWorld(t)
	require.NoError(t, subscribe(w, "a@x.io"))
	require.NoError(t, subscribe(w, "b@x.io"))
	require.NoError(t, appnewsletter.NewConfirmUseCase(w.deps).Execute(ctx, "tok1"))

	all, total, err := appnewsletter.NewListSubscribersUseCase(w.deps).Execute(ctx, appnewsletter.ListInput{Page: 1, PerPage: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, all, 2)

	var csv strings.Builder
	require.NoError(t, appnewsletter.NewExportSubscribersUseCase(w.deps).Execute(ctx, &csv))
	lines := strings.Split(strings.TrimSpace(csv.String()), "\n")
	assert.Equal(t, "email,confirmed_at", lines[0])
	require.Len(t, lines, 2, "only confirmed subscribers are exported")
	assert.True(t, strings.HasPrefix(lines[1], "a@x.io,"))

	id := all[0].ID
	require.NoError(t, appnewsletter.NewEraseSubscriberUseCase(w.deps).Execute(ctx, id))
	assert.NotContains(t, w.repo.byID, id)
	assert.Contains(t, w.outbox.subjects, "newsletter.subscriber.erased")
	assert.ErrorIs(t, appnewsletter.NewEraseSubscriberUseCase(w.deps).Execute(ctx, id), newsletter.ErrSubscriberNotFound)
}

func TestExport_neutralisesSpreadsheetFormulas(t *testing.T) {
	w := newWorld(t)
	s := newsletter.NewSubscriber("tenant-1", "", "=cmd@x.io", "c", "u", w.clock)
	require.NoError(t, s.Confirm(w.clock))
	w.repo.byID["x"] = s

	var csv strings.Builder
	require.NoError(t, appnewsletter.NewExportSubscribersUseCase(w.deps).Execute(ctx, &csv))
	assert.Contains(t, csv.String(), "'=cmd@x.io")
}
