// Package emailintegration wires the admin-only (JWT) Email Integration
// API (see api/emailintegration's doc comment) to its use case.
package emailintegration

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genemail "github.com/adirkuhn/commerceos/api/emailintegration"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type connectEmailUC interface {
	Execute(ctx context.Context, in appintegrations.ConnectEmailInput) (*integrations.Connection, error)
}

type Handler struct {
	connectEmail connectEmailUC
}

func NewHandler(connectEmail connectEmailUC) *Handler {
	return &Handler{connectEmail: connectEmail}
}

var _ genemail.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository, encryptionSecret string) {
	connRepo := repositories.NewConnectionRepo(db)
	h := NewHandler(appintegrations.NewConnectEmailUseCase(db, connRepo, outboxRepo, encryptionSecret))
	genemail.RegisterHandlers(rg, h)
}

func (h *Handler) ConnectEmail(c *gin.Context) {
	var req genemail.ConnectEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
		return
	}
	conn, err := h.connectEmail.Execute(c.Request.Context(), appintegrations.ConnectEmailInput{
		Host: req.Host, Port: req.Port, Username: stringValue(req.Username), Password: stringValue(req.Password), From: req.From,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genemail.Connection{
		Id: conn.ID, Provider: string(conn.Provider), Status: string(conn.Status), CreatedAt: conn.CreatedAt,
	})
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
	case errors.Is(err, integrations.ErrEmailHostRequired), errors.Is(err, integrations.ErrEmailPortRequired),
		errors.Is(err, integrations.ErrEmailFromRequired):
		middleware.FailErr(c, http.StatusUnprocessableEntity, "INVALID_REQUEST", err)
	default:
		log.Printf("emailintegration: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
