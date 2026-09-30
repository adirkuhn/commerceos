package pms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	"github.com/adirkuhn/commerceos/domain/locale"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type setProductTranslationsUC interface {
	Execute(ctx context.Context, productID string, tr map[locale.Locale]pmsDomain.ProductTexts) (*pmsDomain.Product, error)
}

func (h *Handler) SetProductTranslations(c *gin.Context, id genpms.Id) {
	var req genpms.ProductTranslationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	tr := make(map[locale.Locale]pmsDomain.ProductTexts, len(req.Translations))
	for l, t := range req.Translations {
		texts := pmsDomain.ProductTexts{}
		if t.Name != nil {
			texts.Name = *t.Name
		}
		if t.Description != nil {
			texts.Description = *t.Description
		}
		tr[locale.Locale(l)] = texts
	}
	product, err := h.setTranslations.Execute(c.Request.Context(), id, tr)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProduct(*product)
	c.JSON(http.StatusOK, genpms.ProductResponse{Data: &resp})
}

func toGenProductTranslations(tr map[locale.Locale]pmsDomain.ProductTexts) *genpms.ProductTranslations {
	if len(tr) == 0 {
		return nil
	}
	out := make(genpms.ProductTranslations, len(tr))
	for l, t := range tr {
		texts := genpms.ProductTexts{}
		if t.Name != "" {
			name := t.Name
			texts.Name = &name
		}
		if t.Description != "" {
			desc := t.Description
			texts.Description = &desc
		}
		out[string(l)] = texts
	}
	return &out
}
