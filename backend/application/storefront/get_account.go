package storefront

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// resolveSession is shared by every account-scoped use case (GetAccount,
// ListAccountOrders, Logout) -- one place enforcing "session exists, not
// expired, and belongs to this channel" so none of them can drift out of
// sync on the rule.
func resolveSession(ctx context.Context, repo storefrontDomain.CustomerSessionRepository, salesChannelID, token string) (*storefrontDomain.CustomerSession, error) {
	session, err := repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if session == nil || session.SalesChannelID != salesChannelID || session.IsExpired(time.Now()) {
		return nil, storefrontDomain.ErrSessionNotFound
	}
	return session, nil
}

type Account struct {
	Email     string
	FirstName string
	LastName  string
}

type GetAccountUseCase struct {
	db           *gorm.DB
	sessionRepo  storefrontDomain.CustomerSessionRepository
	customerRepo customersDomain.CustomerRepository
}

func NewGetAccountUseCase(db *gorm.DB, sessionRepo storefrontDomain.CustomerSessionRepository, customerRepo customersDomain.CustomerRepository) *GetAccountUseCase {
	return &GetAccountUseCase{db: db, sessionRepo: sessionRepo, customerRepo: customerRepo}
}

func (uc *GetAccountUseCase) Execute(ctx context.Context, salesChannelID, sessionToken string) (*Account, error) {
	var account *Account
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := resolveSession(ctx, uc.sessionRepo, salesChannelID, sessionToken)
		if err != nil {
			return err
		}
		customer, err := uc.customerRepo.FindByID(ctx, session.CustomerID)
		if err != nil {
			if errors.Is(err, customersDomain.ErrCustomerNotFound) {
				return storefrontDomain.ErrSessionNotFound
			}
			return err
		}
		account = &Account{Email: customer.Email, FirstName: customer.FirstName, LastName: customer.LastName}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getAccount: %w", err)
	}
	return account, nil
}
