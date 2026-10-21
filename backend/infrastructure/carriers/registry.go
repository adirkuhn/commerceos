// Package carriers picks the right shipping-label adapter for a connection.
// Each carrier integration (MyParcel, PostNL, ...) registers its own
// domain/integrations.CarrierClientFactory here under its Provider;
// use cases depend only on the Registry, so adding a carrier is a new
// adapter plus one registration line, never a branch in a use case.
package carriers

import (
	"fmt"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// Registry is a CarrierClientFactory that dispatches on the connection's
// Provider to the factory registered for it.
type Registry struct {
	factories map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory
}

var _ integrationsDomain.CarrierClientFactory = Registry{}

func NewRegistry(factories map[integrationsDomain.Provider]integrationsDomain.CarrierClientFactory) Registry {
	return Registry{factories: factories}
}

func (r Registry) NewClient(creds integrationsDomain.CarrierCredentials) (integrationsDomain.CarrierClient, error) {
	factory, ok := r.factories[creds.Provider]
	if !ok {
		return nil, fmt.Errorf("carriers: %q: %w", creds.Provider, integrationsDomain.ErrProviderNotSupported)
	}
	return factory.NewClient(creds)
}

// Supports reports whether provider has a carrier adapter registered --
// i.e. whether its connections can create shipping labels.
func (r Registry) Supports(provider integrationsDomain.Provider) bool {
	_, ok := r.factories[provider]
	return ok
}
