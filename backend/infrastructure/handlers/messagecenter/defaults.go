package messagecenter

import (
	"context"

	"github.com/gin-gonic/gin"

	genmc "github.com/adirkuhn/commerceos/api/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type listTemplateDefaultsUC interface {
	Execute(ctx context.Context) ([]mcDomain.DefaultTemplate, error)
}
type resetTemplateUC interface {
	Execute(ctx context.Context, id string) (*mcDomain.MessageTemplate, error)
}

// WithDefaults adds the default templates gallery and "reset to default".
func (h *Handler) WithDefaults(list listTemplateDefaultsUC, reset resetTemplateUC) *Handler {
	h.listDefaults, h.resetTemplate = list, reset
	return h
}

func (h *Handler) ListMessageTemplateDefaults(c *gin.Context) {
	defaults, err := h.listDefaults.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	out := make([]genmc.MessageTemplateDefault, len(defaults))
	for i, d := range defaults {
		t := d.Template
		out[i] = genmc.MessageTemplateDefault{
			Key: d.Key, TriggerSubject: genmc.TriggerSubject(d.TriggerSubject), Name: t.Name,
			Category: genmc.Category(t.Category), Channels: toGenChannels(t), Subject: t.Subject, Body: t.Body, TextBody: t.TextBody,
		}
	}
	middleware.OK(c, out)
}

func (h *Handler) ResetMessageTemplate(c *gin.Context, id genmc.Id) {
	tmpl, err := h.resetTemplate.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenTemplate(*tmpl))
}
