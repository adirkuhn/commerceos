// Package messagecenter wires the admin-only (JWT) Message Center API (see
// api/messagecenter's doc comment) to its use cases.
package messagecenter

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genmc "github.com/adirkuhn/commerceos/api/messagecenter"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listMessageTemplatesUC interface {
	Execute(ctx context.Context) ([]mcDomain.MessageTemplate, error)
}

type createMessageTemplateUC interface {
	Execute(ctx context.Context, in appmc.CreateMessageTemplateInput) (*mcDomain.MessageTemplate, error)
}

type updateMessageTemplateUC interface {
	Execute(ctx context.Context, in appmc.UpdateMessageTemplateInput) (*mcDomain.MessageTemplate, error)
}

type deleteMessageTemplateUC interface {
	Execute(ctx context.Context, id string) error
}

type listAutomationRulesUC interface {
	Execute(ctx context.Context) ([]mcDomain.AutomationRule, error)
}

type createAutomationRuleUC interface {
	Execute(ctx context.Context, in appmc.CreateAutomationRuleInput) (*mcDomain.AutomationRule, error)
}

type updateAutomationRuleUC interface {
	Execute(ctx context.Context, in appmc.UpdateAutomationRuleInput) (*mcDomain.AutomationRule, error)
}

type deleteAutomationRuleUC interface {
	Execute(ctx context.Context, id string) error
}

type Handler struct {
	listTemplates  listMessageTemplatesUC
	createTemplate createMessageTemplateUC
	updateTemplate updateMessageTemplateUC
	deleteTemplate deleteMessageTemplateUC
	listRules      listAutomationRulesUC
	createRule     createAutomationRuleUC
	updateRule     updateAutomationRuleUC
	deleteRule     deleteAutomationRuleUC
}

func NewHandler(
	list listMessageTemplatesUC, create createMessageTemplateUC, update updateMessageTemplateUC, del deleteMessageTemplateUC,
	listRules listAutomationRulesUC, createRule createAutomationRuleUC, updateRule updateAutomationRuleUC, deleteRule deleteAutomationRuleUC,
) *Handler {
	return &Handler{
		listTemplates: list, createTemplate: create, updateTemplate: update, deleteTemplate: del,
		listRules: listRules, createRule: createRule, updateRule: updateRule, deleteRule: deleteRule,
	}
}

var _ genmc.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	templateRepo := repositories.NewMessageTemplateRepo(db)
	ruleRepo := repositories.NewAutomationRuleRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	h := NewHandler(
		appmc.NewListMessageTemplatesUseCase(db, templateRepo),
		appmc.NewCreateMessageTemplateUseCase(db, templateRepo, outboxRepo),
		appmc.NewUpdateMessageTemplateUseCase(db, templateRepo, outboxRepo),
		appmc.NewDeleteMessageTemplateUseCase(db, templateRepo, outboxRepo),
		appmc.NewListAutomationRulesUseCase(db, ruleRepo),
		appmc.NewCreateAutomationRuleUseCase(db, ruleRepo, templateRepo, outboxRepo),
		appmc.NewUpdateAutomationRuleUseCase(db, ruleRepo, templateRepo, outboxRepo),
		appmc.NewDeleteAutomationRuleUseCase(db, ruleRepo, outboxRepo),
	)
	genmc.RegisterHandlers(rg, h)
}

func (h *Handler) ListMessageTemplates(c *gin.Context) {
	templates, err := h.listTemplates.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genmc.MessageTemplate, len(templates))
	for i, t := range templates {
		out[i] = toGenTemplate(t)
	}
	middleware.OK(c, out)
}

func (h *Handler) CreateMessageTemplate(c *gin.Context) {
	var req genmc.CreateMessageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	tmpl, err := h.createTemplate.Execute(c.Request.Context(), appmc.CreateMessageTemplateInput{
		Name: req.Name, Category: mcDomain.Category(req.Category), Channels: toDomainChannels(req.Channels),
		Subject: stringValue(req.Subject), Body: req.Body,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTemplate(*tmpl))
}

func (h *Handler) UpdateMessageTemplate(c *gin.Context, id genmc.Id) {
	var req genmc.UpdateMessageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	tmpl, err := h.updateTemplate.Execute(c.Request.Context(), appmc.UpdateMessageTemplateInput{
		ID: id, Name: req.Name, Category: mcDomain.Category(req.Category), Channels: toDomainChannels(req.Channels),
		Subject: stringValue(req.Subject), Body: req.Body, Enabled: req.Enabled,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTemplate(*tmpl))
}

func (h *Handler) DeleteMessageTemplate(c *gin.Context, id genmc.Id) {
	err := h.deleteTemplate.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func (h *Handler) ListAutomationRules(c *gin.Context) {
	rules, err := h.listRules.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genmc.AutomationRule, len(rules))
	for i, r := range rules {
		out[i] = toGenRule(r)
	}
	middleware.OK(c, out)
}

func (h *Handler) CreateAutomationRule(c *gin.Context) {
	var req genmc.CreateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	rule, err := h.createRule.Execute(c.Request.Context(), appmc.CreateAutomationRuleInput{
		Name: req.Name, TriggerSubject: string(req.TriggerSubject), Conditions: toDomainConditions(req.Conditions),
		TemplateID: req.TemplateId, ChannelOverride: toDomainChannelPtr(req.ChannelOverride),
		ExtraRecipients: fromGenEmails(req.ExtraRecipients),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenRule(*rule))
}

func (h *Handler) UpdateAutomationRule(c *gin.Context, id genmc.Id) {
	var req genmc.UpdateAutomationRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	rule, err := h.updateRule.Execute(c.Request.Context(), appmc.UpdateAutomationRuleInput{
		ID: id, Name: req.Name, TriggerSubject: string(req.TriggerSubject), Conditions: toDomainConditions(req.Conditions),
		TemplateID: req.TemplateId, ChannelOverride: toDomainChannelPtr(req.ChannelOverride), Enabled: req.Enabled,
		ExtraRecipients: fromGenEmails(req.ExtraRecipients),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenRule(*rule))
}

func (h *Handler) DeleteAutomationRule(c *gin.Context, id genmc.Id) {
	err := h.deleteRule.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenRule(r mcDomain.AutomationRule) genmc.AutomationRule {
	conditions := make([]genmc.Condition, len(r.Conditions))
	for i, c := range r.Conditions {
		conditions[i] = genmc.Condition{Field: c.Field, Operator: genmc.Operator(c.Operator), Value: c.Value}
	}
	var channelOverride *genmc.Channel
	if r.ChannelOverride != nil {
		v := genmc.Channel(*r.ChannelOverride)
		channelOverride = &v
	}
	return genmc.AutomationRule{
		Id: r.ID, Name: r.Name, TriggerSubject: genmc.TriggerSubject(r.TriggerSubject), Conditions: conditions,
		TemplateId: r.TemplateID, ChannelOverride: channelOverride, ExtraRecipients: toGenEmails(r.ExtraRecipients),
		Enabled: r.Enabled, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func fromGenEmails(in *[]openapi_types.Email) []string {
	if in == nil {
		return nil
	}
	out := make([]string, len(*in))
	for i, e := range *in {
		out[i] = string(e)
	}
	return out
}

func toGenEmails(in []string) []openapi_types.Email {
	out := make([]openapi_types.Email, len(in))
	for i, e := range in {
		out[i] = openapi_types.Email(e)
	}
	return out
}

func toDomainConditions(conditions *[]genmc.Condition) []mcDomain.Condition {
	if conditions == nil {
		return nil
	}
	out := make([]mcDomain.Condition, len(*conditions))
	for i, c := range *conditions {
		out[i] = mcDomain.Condition{Field: c.Field, Operator: mcDomain.Operator(c.Operator), Value: c.Value}
	}
	return out
}

func toDomainChannelPtr(ch *genmc.Channel) *messaging.Channel {
	if ch == nil {
		return nil
	}
	v := messaging.Channel(*ch)
	return &v
}

func toGenTemplate(t mcDomain.MessageTemplate) genmc.MessageTemplate {
	channels := make([]genmc.Channel, len(t.Channels))
	for i, ch := range t.Channels {
		channels[i] = genmc.Channel(ch)
	}
	return genmc.MessageTemplate{
		Id: t.ID, Name: t.Name, Category: genmc.Category(t.Category), Channels: channels,
		Subject: t.Subject, Body: t.Body, Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func toDomainChannels(channels []genmc.Channel) []messaging.Channel {
	out := make([]messaging.Channel, len(channels))
	for i, ch := range channels {
		out[i] = messaging.Channel(ch)
	}
	return out
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, mcDomain.ErrTemplateNotFound), errors.Is(err, mcDomain.ErrRuleNotFound):
		middleware.FailErr(c, http.StatusNotFound, "NOT_FOUND", err)
	case errors.Is(err, mcDomain.ErrTemplateNameRequired), errors.Is(err, mcDomain.ErrInvalidCategory),
		errors.Is(err, mcDomain.ErrNoChannelsSelected), errors.Is(err, mcDomain.ErrInvalidChannel),
		errors.Is(err, mcDomain.ErrTemplateBodyRequired), errors.Is(err, mcDomain.ErrRuleNameRequired),
		errors.Is(err, mcDomain.ErrInvalidTriggerSubject), errors.Is(err, mcDomain.ErrTemplateIDRequired),
		errors.Is(err, mcDomain.ErrInvalidCondition), errors.Is(err, mcDomain.ErrInvalidExtraRecipient),
		errors.Is(err, mcDomain.ErrTooManyExtraRecipients):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
	default:
		log.Printf("messagecenter: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
