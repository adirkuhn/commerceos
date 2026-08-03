package notifications

import "time"

// Type is intentionally open-ended (plain string, not a closed enum) —
// notifications are a general concept any feature can produce into. Import
// jobs are the first producer (see application/channels/create_import_job.go
// and cmd/scheduler's job runner); the type just needs to be unique enough
// for the frontend to pick an icon/style by prefix if it wants to.
type Type string

const (
	TypeImportStarted   Type = "import.started"
	TypeImportCompleted Type = "import.completed"
	TypeImportFailed    Type = "import.failed"
)

type Notification struct {
	ID        string
	TenantID  string
	Type      Type
	Title     string
	Message   string
	Read      bool
	CreatedAt time.Time
}
