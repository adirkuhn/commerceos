package theme

import "time"

type VersionState string

const (
	VersionStateDraft  VersionState = "draft"
	VersionStateSealed VersionState = "sealed"
)

// Version is a snapshot of a theme: its settings plus its documents
// (stored separately, see Document). A theme has at most one draft -- the
// only mutable version. Publishing seals the draft for good; the next edit
// starts a new draft copied from the published one (NextDraftFrom).
//
// Revision is the draft's optimistic-concurrency counter: every write
// states the revision it was based on, so two tabs or two people editing
// the same draft can't silently overwrite each other.
type Version struct {
	ID        string
	TenantID  string
	ThemeID   string
	Number    int
	State     VersionState
	Revision  int
	Settings  Settings
	Note      string
	CreatedBy string
	CreatedAt time.Time
	SealedAt  *time.Time
}

func NewDraftVersion(t Theme, number int, settings Settings, createdBy string, now time.Time) Version {
	return Version{
		TenantID: t.TenantID, ThemeID: t.ID, Number: number, State: VersionStateDraft, Revision: 1,
		Settings: settings, CreatedBy: createdBy, CreatedAt: now,
	}
}

// NextDraftFrom starts a new draft from a published version -- same
// settings, the documents are copied alongside by the caller.
func NextDraftFrom(published Version, number int, createdBy string, now time.Time) Version {
	return Version{
		TenantID: published.TenantID, ThemeID: published.ThemeID, Number: number, State: VersionStateDraft,
		Revision: 1, Settings: published.Settings, CreatedBy: createdBy, CreatedAt: now,
	}
}

// KeepPublishedVersions is how many published versions of a theme are
// kept for rollback; older ones are pruned on publish.
const KeepPublishedVersions = 20

func (v Version) IsDraft() bool { return v.State == VersionStateDraft }

// BeginEdit guards every draft write: the draft must still be a draft and
// the caller must have seen its latest revision. On success the revision
// moves on, so the next writer must have seen this edit.
func (v *Version) BeginEdit(expectedRevision int) error {
	if !v.IsDraft() {
		return ErrVersionSealed
	}
	if v.Revision != expectedRevision {
		return ErrRevisionConflict
	}
	v.Revision++
	return nil
}

// Publish seals the draft and makes it the theme's live version. Nothing
// changes when any check fails.
func Publish(t *Theme, draft *Version, expectedRevision int, note string, now time.Time) error {
	if draft.ThemeID != t.ID {
		return ErrVersionNotInTheme
	}
	if !draft.IsDraft() {
		return ErrVersionSealed
	}
	if draft.Revision != expectedRevision {
		return ErrRevisionConflict
	}
	sealedAt := now
	draft.State = VersionStateSealed
	draft.SealedAt = &sealedAt
	draft.Note = note
	id := draft.ID
	t.PublishedVersionID = &id
	t.IsLive = true
	t.UpdatedAt = now
	return nil
}

// Rollback makes an older published version live again. No copy is made:
// published versions are immutable, so pointing at one is enough.
func Rollback(t *Theme, target Version, now time.Time) error {
	if target.ThemeID != t.ID {
		return ErrVersionNotInTheme
	}
	if target.State != VersionStateSealed {
		return ErrVersionNotSealed
	}
	id := target.ID
	t.PublishedVersionID = &id
	t.IsLive = true
	t.UpdatedAt = now
	return nil
}
