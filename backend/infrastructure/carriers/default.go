package carriers

import (
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/myparcel"
)

// NewDefaultRegistry is every carrier integration this build ships with --
// the one place a new carrier adapter gets registered.
func NewDefaultRegistry() Registry {
	return NewRegistry(map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory{
		integrationsDomain.ProviderMyParcel: myparcel.NewCarrierClientFactory(),
	})
}
