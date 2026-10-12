package messagecenter

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genmc "github.com/adirkuhn/commerceos/api/messagecenter"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type listSentEmailsUC interface {
	Execute(ctx context.Context, f mcDomain.SentEmailFilter) (*appmc.SentEmailPage, error)
}
type getSentEmailUC interface {
	Execute(ctx context.Context, id string) (*mcDomain.SentEmail, error)
}
type deleteSentEmailsUC interface {
	Execute(ctx context.Context, ids []string) (int, error)
}

// WithSentEmails adds the sent-email history routes.
func (h *Handler) WithSentEmails(list listSentEmailsUC, get getSentEmailUC, del deleteSentEmailsUC) *Handler {
	h.listSentEmails, h.getSentEmail, h.deleteSentEmails = list, get, del
	return h
}

func (h *Handler) ListSentEmails(c *gin.Context, params genmc.ListSentEmailsParams) {
	f := mcDomain.SentEmailFilter{From: params.From, To: params.To}
	if params.Status != nil {
		f.Status = mcDomain.SentEmailStatus(*params.Status)
	}
	if params.Source != nil {
		f.Source = mcDomain.SentEmailSource(*params.Source)
	}
	if params.Q != nil {
		f.Query = *params.Q
	}
	if params.Page != nil {
		f.Page = *params.Page
	}
	if params.PerPage != nil {
		f.PerPage = *params.PerPage
	}
	page, err := h.listSentEmails.Execute(c.Request.Context(), f)
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]genmc.SentEmailSummary, len(page.Items))
	for i, e := range page.Items {
		items[i] = toGenSentEmailSummary(e)
	}
	middleware.OK(c, genmc.SentEmailList{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage})
}

func (h *Handler) GetSentEmail(c *gin.Context, id genmc.Id) {
	e, err := h.getSentEmail.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	s := toGenSentEmailSummary(*e)
	middleware.OK(c, genmc.SentEmail{
		Id: s.Id, Source: s.Source, Recipient: s.Recipient, Subject: s.Subject, Status: s.Status, Reason: s.Reason,
		RuleId: s.RuleId, TemplateId: s.TemplateId, TriggerSubject: s.TriggerSubject, CreatedAt: s.CreatedAt, HtmlBody: e.HTMLBody,
	})
}

func (h *Handler) DeleteSentEmails(c *gin.Context) {
	var req genmc.DeleteSentEmailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	n, err := h.deleteSentEmails.Execute(c.Request.Context(), req.Ids)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, gin.H{"deleted": n})
}

func (h *Handler) DeleteSentEmail(c *gin.Context, id genmc.Id) {
	_, err := h.deleteSentEmails.Execute(c.Request.Context(), []string{id})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenSentEmailSummary(e mcDomain.SentEmail) genmc.SentEmailSummary {
	return genmc.SentEmailSummary{
		Id: e.ID, Source: genmc.SentEmailSource(e.Source), Recipient: e.Recipient, Subject: e.Subject, Status: genmc.SentEmailStatus(e.Status),
		Reason: e.Reason, RuleId: e.RuleID, TemplateId: e.TemplateID, TriggerSubject: e.TriggerSubject, CreatedAt: e.CreatedAt,
	}
}
