package messagecenter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genmc "github.com/adirkuhn/commerceos/api/messagecenter"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	mchandler "github.com/adirkuhn/commerceos/infrastructure/handlers/messagecenter"
)

type stubListSent struct {
	out *appmc.SentEmailPage
	err error
	got mcDomain.SentEmailFilter
}

func (s *stubListSent) Execute(_ context.Context, f mcDomain.SentEmailFilter) (*appmc.SentEmailPage, error) {
	s.got = f
	return s.out, s.err
}

type stubGetSent struct {
	out *mcDomain.SentEmail
	err error
}

func (s *stubGetSent) Execute(context.Context, string) (*mcDomain.SentEmail, error) {
	return s.out, s.err
}

type stubDeleteSent struct {
	n   int
	err error
	got []string
}

func (s *stubDeleteSent) Execute(_ context.Context, ids []string) (int, error) {
	s.got = ids
	return s.n, s.err
}

func sentHandler(list *stubListSent, get *stubGetSent, del *stubDeleteSent) *mchandler.Handler {
	return newHandler(nil, nil, nil, nil, nil, nil, nil, nil).WithSentEmails(list, get, del)
}

var sentAt = time.Date(2026, 10, 12, 9, 30, 0, 0, time.UTC)

func TestListSentEmailsHandler_passesFiltersAndReturnsThePage(t *testing.T) {
	list := &stubListSent{out: &appmc.SentEmailPage{Total: 3, Page: 2, PerPage: 1, Items: []mcDomain.SentEmail{{
		ID: "e1", Source: mcDomain.SentEmailSourceAutomation, Recipient: "ana@example.com", Subject: "Order", Status: mcDomain.SentEmailSkipped,
		Reason: mcDomain.SkipNoRecipient, RuleID: "r1", TemplateID: "t1", TriggerSubject: "storefront.order.placed", CreatedAt: sentAt,
	}}}}

	w := doRequest(newTestRouter(sentHandler(list, nil, nil)), http.MethodGet,
		"/api/v1/sent-emails?status=skipped&source=automation&q=ana&from=2026-10-01T00:00:00Z&page=2&per_page=1", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, mcDomain.SentEmailSkipped, list.got.Status)
	assert.Equal(t, mcDomain.SentEmailSourceAutomation, list.got.Source)
	assert.Equal(t, "ana", list.got.Query)
	require.NotNil(t, list.got.From)
	assert.Nil(t, list.got.To)
	assert.Equal(t, 2, list.got.Page)
	assert.Equal(t, 1, list.got.PerPage)
	var body struct {
		Data genmc.SentEmailList `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, 3, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, genmc.SentEmailSummary{
		Id: "e1", Source: genmc.Automation, Recipient: "ana@example.com", Subject: "Order", Status: genmc.Skipped,
		Reason: "no_recipient", RuleId: "r1", TemplateId: "t1", TriggerSubject: "storefront.order.placed", CreatedAt: sentAt,
	}, body.Data.Items[0])
}

func TestListSentEmailsHandler_badFilterIs422(t *testing.T) {
	list := &stubListSent{err: fmt.Errorf("listSentEmails: %w", mcDomain.ErrSentEmailFilter)}
	w := doRequest(newTestRouter(sentHandler(list, nil, nil)), http.MethodGet, "/api/v1/sent-emails", "")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "SENT_EMAIL_INVALID_FILTER")
}

func TestGetSentEmailHandler(t *testing.T) {
	get := &stubGetSent{out: &mcDomain.SentEmail{ID: "e1", HTMLBody: "<p>Hi</p>", Status: mcDomain.SentEmailSent, CreatedAt: sentAt}}
	w := doRequest(newTestRouter(sentHandler(nil, get, nil)), http.MethodGet, "/api/v1/sent-emails/e1", "")

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data genmc.SentEmail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "<p>Hi</p>", body.Data.HtmlBody)

	missing := &stubGetSent{err: fmt.Errorf("getSentEmail: %w", mcDomain.ErrSentEmailNotFound)}
	w = doRequest(newTestRouter(sentHandler(nil, missing, nil)), http.MethodGet, "/api/v1/sent-emails/x", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "SENT_EMAIL_NOT_FOUND")
}

func TestDeleteSentEmailsHandler(t *testing.T) {
	del := &stubDeleteSent{n: 2}
	w := doRequest(newTestRouter(sentHandler(nil, nil, del)), http.MethodPost, "/api/v1/sent-emails/bulk-delete", `{"ids":["a","b"]}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"a", "b"}, del.got)
	assert.JSONEq(t, `{"data":{"deleted":2},"error":null}`, w.Body.String())

	w = doRequest(newTestRouter(sentHandler(nil, nil, &stubDeleteSent{})), http.MethodPost, "/api/v1/sent-emails/bulk-delete", `{`)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteSentEmailHandler(t *testing.T) {
	del := &stubDeleteSent{n: 1}
	w := doRequest(newTestRouter(sentHandler(nil, nil, del)), http.MethodDelete, "/api/v1/sent-emails/a", "")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"a"}, del.got)

	gone := &stubDeleteSent{err: fmt.Errorf("deleteSentEmails: %w", mcDomain.ErrSentEmailNotFound)}
	w = doRequest(newTestRouter(sentHandler(nil, nil, gone)), http.MethodDelete, "/api/v1/sent-emails/a", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}
