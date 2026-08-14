package channels

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// maxCustomerImportPages mirrors maxOrderImportPages -- a real store's
// customer list can be as large as its order history.
const maxCustomerImportPages = 500

type ImportWooCustomersInput struct {
	ConnectionID string
}

type ImportCustomersResult struct {
	Created int
	// Skipped counts a Woo customer already linked to a CommerceOS
	// Customer row (WooCustomerID match) -- this use case never updates
	// an existing customer, only creates new ones, so a re-run is a safe
	// no-op.
	Skipped int
	Errors  []string
}

// ImportWooCustomersUseCase pulls every customer account from a
// WooCommerce store into domain/customers.Customer. Always a full scan
// (Woo's customers endpoint has no modified-since filter worth building
// incremental sync around) -- used only by the one-time bootstrap that
// runs alongside a connection's first products import (see
// RunManualWooImportUseCase), not on any recurring schedule. Passwords are
// deliberately never touched: Woo's REST API never exposes password
// hashes to begin with, and this codebase has no local password-checking
// code for an imported hash to be verified against even if it did --
// CommerceOS auth is delegated entirely to lm2.
type ImportWooCustomersUseCase struct {
	db               *gorm.DB
	connRepo         integrationsDomain.ConnectionRepository
	wooFactory       integrationsDomain.WooClientFactory
	encryptionSecret string
	customerRepo     customersDomain.CustomerRepository
}

func NewImportWooCustomersUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	customerRepo customersDomain.CustomerRepository,
) *ImportWooCustomersUseCase {
	return &ImportWooCustomersUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory,
		encryptionSecret: encryptionSecret, customerRepo: customerRepo,
	}
}

func (uc *ImportWooCustomersUseCase) Execute(ctx context.Context, in ImportWooCustomersInput) (*ImportCustomersResult, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *existing
		secret, err = crypto.Decrypt(uc.encryptionSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("importWooCustomers: %w", err)
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("importWooCustomers: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	result := ImportCustomersResult{}
	for page := 1; page <= maxCustomerImportPages; page++ {
		customers, err := client.ListCustomers(ctx, page, 50)
		if err != nil {
			return nil, fmt.Errorf("importWooCustomers: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
		}
		if len(customers) == 0 {
			break
		}
		for _, woo := range customers {
			uc.importOne(ctx, woo, &result)
		}
	}
	return &result, nil
}

func (uc *ImportWooCustomersUseCase) importOne(ctx context.Context, woo integrationsDomain.WooCustomer, result *ImportCustomersResult) {
	wooID := int64(woo.ID)

	var existing *customersDomain.Customer
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		existing, err = uc.customerRepo.FindByWooCustomerID(ctx, wooID)
		return err
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("woo customer %d: %v", woo.ID, err))
		return
	}
	if existing != nil {
		result.Skipped++
		return
	}

	now := time.Now()
	dateCreated, err := parseWooGMTTime(woo.DateCreated)
	if err != nil {
		dateCreated = now
	}
	dateModified, err := parseWooGMTTime(woo.DateModified)
	if err != nil {
		dateModified = dateCreated
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		customer := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), WooCustomerID: &wooID,
			Email: woo.Email, FirstName: woo.FirstName, LastName: woo.LastName,
			Username: woo.Username, Role: woo.Role,
			Billing: wooAddressToCustomerAddress(woo.Billing), Shipping: wooAddressToCustomerAddress(woo.Shipping),
			IsPayingCustomer: woo.IsPayingCustomer, AvatarURL: woo.AvatarURL,
			DateCreated: dateCreated, DateModified: dateModified,
		}
		return uc.customerRepo.Create(ctx, &customer)
	})
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("woo customer %d: %v", woo.ID, err))
		return
	}
	result.Created++
}

func wooAddressToCustomerAddress(a integrationsDomain.WooAddress) customersDomain.Address {
	return customersDomain.Address{
		FirstName: a.FirstName, LastName: a.LastName, Company: a.Company,
		Address1: a.Address1, Address2: a.Address2, City: a.City, State: a.State,
		Postcode: a.Postcode, Country: a.Country, Email: a.Email, Phone: a.Phone,
	}
}
