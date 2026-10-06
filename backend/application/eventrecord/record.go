// Package eventrecord is how a use case raises the standard action event
// (events.EntityAction): one Record call inside the use case's
// tenant.WithTenant transaction writes it to the outbox, atomic with the
// change itself. Every state-changing use case raises one (docs/events.md).
package eventrecord

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/idgen"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// Enqueuer is the outbox (domain/outbox.Repository).
type Enqueuer interface {
	Enqueue(ctx context.Context, subject, eventID string, payload []byte, occurredAt time.Time) error
}

// Option adds detail to a recorded action: Data or Changes.
type Option interface{ apply(*events.EntityAction) }

// Data is extra facts about the action (e.g. source, quantity).
type Data map[string]any

func (d Data) apply(e *events.EntityAction) { e.Data = d }

// Changes is the fields an update changed, before and after.
type Changes struct{ fields map[string]events.FieldChange }

func NewChanges() Changes { return Changes{fields: map[string]events.FieldChange{}} }

// Set records field's change, only when before and after differ
// (pointers compare by the value they point to).
func (c Changes) Set(field string, before, after any) Changes {
	b, a := deref(before), deref(after)
	if !reflect.DeepEqual(b, a) {
		c.fields[field] = events.FieldChange{Before: b, After: a}
	}
	return c
}

// Fields lists the changed fields, sorted.
func (c Changes) Fields() []string {
	out := make([]string, 0, len(c.fields))
	for f := range c.fields {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}

func (c Changes) apply(e *events.EntityAction) {
	if len(c.fields) > 0 {
		e.Changes = c.fields
	}
}

func deref(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		return v
	}
	if rv.IsNil() {
		return nil
	}
	return rv.Elem().Interface()
}

// Record writes action on entityID to the outbox. Must run inside
// tenant.WithTenant (the outbox takes the transaction from ctx).
func Record(ctx context.Context, outbox Enqueuer, action events.ActionType, entityID string, opts ...Option) error {
	e := events.EntityAction{
		Base:     events.Base{EventID: idgen.NewV4(), OccurredAt: time.Now().UTC()},
		Type:     action,
		TenantID: tenant.IDFromContext(ctx),
		Actor:    actorFrom(ctx),
		EntityID: entityID,
	}
	for _, o := range opts {
		o.apply(&e)
	}
	payload, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("record %s: %w", action.Subject, err)
	}
	if err := outbox.Enqueue(ctx, action.Subject, e.EventID, payload, e.OccurredAt); err != nil {
		return fmt.Errorf("record %s: %w", action.Subject, err)
	}
	return nil
}

func actorFrom(ctx context.Context) events.Actor {
	userID := tenant.UserIDFromContext(ctx)
	if userID == "" {
		return events.Actor{Type: events.ActorSystem}
	}
	return events.Actor{Type: events.ActorUser, UserID: userID, Role: string(tenant.RoleFromContext(ctx))}
}
