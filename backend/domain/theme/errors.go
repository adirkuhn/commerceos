package theme

import "errors"

// ErrInvalidBlockType guards Upsert -- every Block.Type must be one of the
// fixed BlockType values the storefront-frontend actually knows how to
// render.
var ErrInvalidBlockType = errors.New("invalid theme block type")
