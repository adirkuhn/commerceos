package theme

import "errors"

// ErrInvalidBlockType guards Upsert -- every Block.Type must be one of the
// fixed BlockType values the storefront-frontend actually knows how to
// render.
var ErrInvalidBlockType = errors.New("invalid theme block type")

// ErrInvalidPageSlug guards Upsert -- Slug must be one of the fixed
// PageSlug values the storefront-frontend actually has a route for.
var ErrInvalidPageSlug = errors.New("invalid theme page slug")

// ErrInvalidEditorMode guards Upsert -- EditorMode must be Freeform or Puck.
var ErrInvalidEditorMode = errors.New("invalid theme editor mode")

// ErrEditSessionNotFound covers both "no such token" and "token expired" --
// see EditSessionRepository.FindByToken's own doc comment on why those
// stay indistinguishable to the caller.
var ErrEditSessionNotFound = errors.New("theme edit session not found or expired")
