package integrations_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

// stubBolClient implements only what ListBolEconomicOperatorsUseCase needs
// (ListEconomicOperators) -- every other BolClient method is unreachable
// from this use case, so it panics if called, same "fail loud on
// unexpected use" convention as this package's other narrow test doubles.
type stubBolClient struct {
	operators []integrationsDomain.BolEconomicOperator
	err       error
}

func (s *stubBolClient) CreateOffer(context.Context, integrationsDomain.BolOfferInput) (string, error) {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) UpdateOffer(context.Context, string, integrationsDomain.BolOfferInput) error {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) IsUnknownEAN(error) bool {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) SubmitProductContent(context.Context, integrationsDomain.BolProductContentInput) error {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) GetCommission(context.Context, string, float64) (integrationsDomain.BolCommission, error) {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) ListOrders(context.Context, int, string) ([]integrationsDomain.BolOrder, error) {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) GetOrder(context.Context, string) (*integrationsDomain.BolOrder, error) {
	panic("not used by ListBolEconomicOperatorsUseCase")
}
func (s *stubBolClient) ListEconomicOperators(context.Context) ([]integrationsDomain.BolEconomicOperator, error) {
	return s.operators, s.err
}

type stubBolClientFactory struct{ client integrationsDomain.BolClient }

func (f *stubBolClientFactory) NewClient(_, _ string) (integrationsDomain.BolClient, error) {
	return f.client, nil
}

func bolTestConn() integrationsDomain.Connection {
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "secret")
	if err != nil {
		panic(err)
	}
	return integrationsDomain.Connection{
		ID: "conn-1", Provider: integrationsDomain.ProviderBol,
		ConsumerKey: "client-id", ConsumerSecretEncrypted: encrypted,
	}
}

func TestListBolEconomicOperators_returnsOperatorsFromBol(t *testing.T) {
	conn := bolTestConn()
	connRepo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) {
			require.Equal(t, "conn-1", id)
			return &conn, nil
		},
	}
	client := &stubBolClient{operators: []integrationsDomain.BolEconomicOperator{
		{ID: "3fa85f64-5717-4562-b3fc-2c963f66afa6", Name: "Cantinho Brasileiro", Status: "VALID"},
	}}
	uc := appintegrations.NewListBolEconomicOperatorsUseCase(newMockGormDB(t), connRepo, &stubBolClientFactory{client: client}, testEncryptionSecret)

	operators, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Len(t, operators, 1)
	require.Equal(t, "3fa85f64-5717-4562-b3fc-2c963f66afa6", operators[0].ID)
	require.Equal(t, "VALID", operators[0].Status)
}

func TestListBolEconomicOperators_propagatesBolAPIError(t *testing.T) {
	conn := bolTestConn()
	connRepo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return &conn, nil },
	}
	client := &stubBolClient{err: errors.New("bol: request failed (403): missing scope economic-operators.read")}
	uc := appintegrations.NewListBolEconomicOperatorsUseCase(newMockGormDB(t), connRepo, &stubBolClientFactory{client: client}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), "conn-1")

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrBolEconomicOperatorsRequestFailed))
	require.ErrorContains(t, err, "403")
}

func TestListBolEconomicOperators_propagatesConnectionNotFound(t *testing.T) {
	connRepo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewListBolEconomicOperatorsUseCase(newMockGormDB(t), connRepo, &stubBolClientFactory{}, testEncryptionSecret)

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
