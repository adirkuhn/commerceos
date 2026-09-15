// Package whatsapp wires the admin-only (JWT) WhatsApp conversations API
// (see api/whatsapp's doc comment) to its use cases.
package whatsapp

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genwhatsapp "github.com/adirkuhn/commerceos/api/whatsapp"
	appwhatsapp "github.com/adirkuhn/commerceos/application/whatsapp"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listConversationsUC interface {
	Execute(ctx context.Context) ([]whatsappDomain.ConversationSummary, error)
}

type listConversationMessagesUC interface {
	Execute(ctx context.Context, phoneE164 string) ([]whatsappDomain.Message, error)
}

type deleteConversationUC interface {
	Execute(ctx context.Context, phoneE164 string) error
}

type Handler struct {
	listConversations        listConversationsUC
	listConversationMessages listConversationMessagesUC
	deleteConversation       deleteConversationUC
}

func NewHandler(listConversations listConversationsUC, listConversationMessages listConversationMessagesUC, deleteConversation deleteConversationUC) *Handler {
	return &Handler{listConversations: listConversations, listConversationMessages: listConversationMessages, deleteConversation: deleteConversation}
}

var _ genwhatsapp.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	messageRepo := repositories.NewWhatsAppMessageRepo(db)
	h := NewHandler(
		appwhatsapp.NewListWhatsAppConversationsUseCase(db, messageRepo),
		appwhatsapp.NewListWhatsAppMessagesByPhoneUseCase(db, messageRepo),
		appwhatsapp.NewDeleteWhatsAppConversationUseCase(db, messageRepo),
	)
	genwhatsapp.RegisterHandlers(rg, h)
}

func (h *Handler) ListWhatsAppConversations(c *gin.Context) {
	conversations, err := h.listConversations.Execute(c.Request.Context())
	if err != nil {
		log.Printf("whatsapp: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	out := make([]genwhatsapp.WhatsAppConversation, len(conversations))
	for i, conv := range conversations {
		out[i] = toGenConversation(conv)
	}
	middleware.OK(c, out)
}

func (h *Handler) ListWhatsAppConversationMessages(c *gin.Context, phone string) {
	messages, err := h.listConversationMessages.Execute(c.Request.Context(), phone)
	if err != nil {
		log.Printf("whatsapp: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	out := make([]genwhatsapp.WhatsAppConversationMessage, len(messages))
	for i, m := range messages {
		out[i] = toGenConversationMessage(m)
	}
	middleware.OK(c, out)
}

func (h *Handler) DeleteWhatsAppConversation(c *gin.Context, phone string) {
	if err := h.deleteConversation.Execute(c.Request.Context(), phone); err != nil {
		log.Printf("whatsapp: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	middleware.OK(c, nil)
}

func toGenConversation(conv whatsappDomain.ConversationSummary) genwhatsapp.WhatsAppConversation {
	return genwhatsapp.WhatsAppConversation{
		PhoneE164: conv.PhoneE164, CustomerId: conv.CustomerID, CustomerName: &conv.CustomerName,
		LastDirection: genwhatsapp.WhatsAppConversationLastDirection(conv.LastDirection),
		LastBody:      conv.LastBody, LastMessageAt: conv.LastMessageAt,
	}
}

func toGenConversationMessage(m whatsappDomain.Message) genwhatsapp.WhatsAppConversationMessage {
	return genwhatsapp.WhatsAppConversationMessage{
		Id: m.ID, PhoneE164: m.PhoneE164, CustomerId: m.CustomerID,
		Direction: genwhatsapp.WhatsAppConversationMessageDirection(m.Direction),
		Body:      m.Body, CreatedAt: m.CreatedAt,
	}
}
