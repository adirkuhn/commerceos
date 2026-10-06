package whatsapp

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"

	"gorm.io/gorm"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ListWhatsAppConversationsUseCase powers the "WhatsApp Web"-style admin
// conversations list -- one row per phone number, newest conversation
// first.
type ListWhatsAppConversationsUseCase struct {
	db          *gorm.DB
	messageRepo whatsappDomain.MessageRepository
}

func NewListWhatsAppConversationsUseCase(db *gorm.DB, messageRepo whatsappDomain.MessageRepository) *ListWhatsAppConversationsUseCase {
	return &ListWhatsAppConversationsUseCase{db: db, messageRepo: messageRepo}
}

func (uc *ListWhatsAppConversationsUseCase) Execute(ctx context.Context) ([]whatsappDomain.ConversationSummary, error) {
	var out []whatsappDomain.ConversationSummary
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.messageRepo.ListConversations(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listWhatsAppConversations: %w", err)
	}
	return out, nil
}

// ListWhatsAppMessagesByPhoneUseCase powers the conversations list's thread
// view -- every message for one phone number, newest first, regardless of
// whether it ever matched a customer.
type ListWhatsAppMessagesByPhoneUseCase struct {
	db          *gorm.DB
	messageRepo whatsappDomain.MessageRepository
}

func NewListWhatsAppMessagesByPhoneUseCase(db *gorm.DB, messageRepo whatsappDomain.MessageRepository) *ListWhatsAppMessagesByPhoneUseCase {
	return &ListWhatsAppMessagesByPhoneUseCase{db: db, messageRepo: messageRepo}
}

func (uc *ListWhatsAppMessagesByPhoneUseCase) Execute(ctx context.Context, phoneE164 string) ([]whatsappDomain.Message, error) {
	var out []whatsappDomain.Message
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.messageRepo.ListByPhone(ctx, phoneE164)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listWhatsAppMessagesByPhone: %w", err)
	}
	return out, nil
}

// DeleteWhatsAppConversationUseCase powers the conversations list's
// "Delete chat" action -- permanently removes every message for one phone
// number.
type DeleteWhatsAppConversationUseCase struct {
	db          *gorm.DB
	messageRepo whatsappDomain.MessageRepository
	outbox      eventrecord.Enqueuer
}

func NewDeleteWhatsAppConversationUseCase(db *gorm.DB, messageRepo whatsappDomain.MessageRepository, outbox eventrecord.Enqueuer) *DeleteWhatsAppConversationUseCase {
	return &DeleteWhatsAppConversationUseCase{db: db, messageRepo: messageRepo, outbox: outbox}
}

func (uc *DeleteWhatsAppConversationUseCase) Execute(ctx context.Context, phoneE164 string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.messageRepo.DeleteByPhone(ctx, phoneE164); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.WhatsAppConversationDeleted, phoneE164)
	})
	if err != nil {
		return fmt.Errorf("deleteWhatsAppConversation: %w", err)
	}
	return nil
}
