package customers

import (
	"context"
	"fmt"

	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// ListCustomerWhatsAppMessagesUseCase powers the customer detail page's
// WhatsApp conversation panel -- every message (inbound and outbound
// reply) recorded against this customer, newest first.
type ListCustomerWhatsAppMessagesUseCase struct {
	db          *gorm.DB
	messageRepo whatsappDomain.MessageRepository
}

func NewListCustomerWhatsAppMessagesUseCase(db *gorm.DB, messageRepo whatsappDomain.MessageRepository) *ListCustomerWhatsAppMessagesUseCase {
	return &ListCustomerWhatsAppMessagesUseCase{db: db, messageRepo: messageRepo}
}

func (uc *ListCustomerWhatsAppMessagesUseCase) Execute(ctx context.Context, customerID string) ([]whatsappDomain.Message, error) {
	var out []whatsappDomain.Message
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		out, err = uc.messageRepo.ListByCustomer(ctx, customerID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listCustomerWhatsAppMessages: %w", err)
	}
	return out, nil
}
