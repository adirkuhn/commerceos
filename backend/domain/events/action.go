package events

// ActionType is one kind of standard action event: a subject following
// the naming standard (naming.go) plus what it means, for the catalog
// (docs/events.md). Declared once per action in actions_<domain>.go.
type ActionType struct {
	Subject     string
	Description string
}

var actionTypes []ActionType

func defineAction(subject, description string) ActionType {
	a := ActionType{Subject: subject, Description: description}
	actionTypes = append(actionTypes, a)
	return a
}

// ActionTypes lists every registered action type.
func ActionTypes() []ActionType { return append([]ActionType(nil), actionTypes...) }

func FindActionType(subject string) (ActionType, bool) {
	for _, a := range actionTypes {
		if a.Subject == subject {
			return a, true
		}
	}
	return ActionType{}, false
}

type ActorType string

const (
	// ActorUser: a person, through the admin API (JWT user_id).
	ActorUser ActorType = "user"
	// ActorSystem: a scheduler job, event consumer or integration.
	ActorSystem ActorType = "system"
)

// Actor is who caused the action.
type Actor struct {
	Type   ActorType `json:"type"`
	UserID string    `json:"user_id,omitempty"`
	Role   string    `json:"role,omitempty"`
}

// FieldChange is one field's value before and after an update.
type FieldChange struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

// EntityAction is the standard event every state-changing action raises:
// who (Actor) did what (Type) to which record (EntityID) of a tenant, with
// the fields it changed (Changes, before/after -- updates) and any extra
// facts (Data). Consumers needing the full record read it by EntityID.
type EntityAction struct {
	Base
	Type     ActionType             `json:"-"`
	TenantID string                 `json:"tenant_id"`
	Actor    Actor                  `json:"actor"`
	EntityID string                 `json:"entity_id"`
	Changes  map[string]FieldChange `json:"changes,omitempty"`
	Data     map[string]any         `json:"data,omitempty"`
}

func (e EntityAction) EventName() string { return e.Type.Subject }

// MarshalJSON adds the subject as "event" so a payload describes itself.
func (e EntityAction) MarshalJSON() ([]byte, error) {
	type plain EntityAction
	return marshalWithEvent(e.Type.Subject, plain(e))
}
