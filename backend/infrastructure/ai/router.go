package ai

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// The tenant's AI connections form an ordered list and every AI feature
// routes through it like a router: the first enabled connection is tried,
// and any failure -- busy, out of quota, bad key, bad output -- moves on to
// the next. Only when every connection failed does the feature fail, with
// each one's reason.

// routeTarget is one connection's adapter, named for logs and errors.
type routeTarget[T any] struct {
	name   string
	target T
}

func route[T, R any](ctx context.Context, targets []routeTarget[T], call func(T) (R, error)) (R, error) {
	var zero R
	var failures []error
	for i, t := range targets {
		out, err := call(t.target)
		if err == nil {
			return out, nil
		}
		failures = append(failures, fmt.Errorf("%s: %w", t.name, err))
		if ctx.Err() != nil {
			break
		}
		if i < len(targets)-1 {
			log.Printf("ai router: %s failed, trying %s: %v", t.name, targets[i+1].name, err)
		}
	}
	return zero, errors.Join(failures...)
}

// loadedConnection is an enabled connection with its key decrypted.
type loadedConnection struct {
	conn   integrationsDomain.AIConnection
	apiKey string
}

// connectionName is how a connection appears in logs and errors.
func connectionName(c integrationsDomain.AIConnection) string {
	if c.Label != "" {
		return fmt.Sprintf("%s (%s)", c.Label, c.Provider)
	}
	return fmt.Sprintf("%s #%d", c.Provider, c.Position+1)
}

// loadConnections reads the tenant's enabled connections in routing order.
// One whose key can't be decrypted is skipped (logged), not fatal: the
// others can still serve.
func loadConnections(ctx context.Context, db *gorm.DB, repo integrationsDomain.AIConnectionRepository, secret string) ([]loadedConnection, error) {
	var all []integrationsDomain.AIConnection
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		all, err = repo.List(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	var out []loadedConnection
	for _, c := range all {
		if !c.Enabled {
			continue
		}
		key := ""
		if c.APIKeyEncrypted != "" {
			if key, err = crypto.Decrypt(secret, c.APIKeyEncrypted); err != nil {
				log.Printf("ai router: %s skipped, key can't be decrypted: %v", connectionName(c), err)
				continue
			}
		}
		out = append(out, loadedConnection{conn: c, apiKey: key})
	}
	return out, nil
}

// buildTargets builds each connection's adapter with build; a connection
// build refuses (an unknown provider) is skipped and logged.
func buildTargets[T any](conns []loadedConnection, build func(loadedConnection) (T, error)) []routeTarget[T] {
	var out []routeTarget[T]
	for _, lc := range conns {
		t, err := build(lc)
		if err != nil {
			log.Printf("ai router: %s skipped: %v", connectionName(lc.conn), err)
			continue
		}
		out = append(out, routeTarget[T]{name: connectionName(lc.conn), target: t})
	}
	return out
}
