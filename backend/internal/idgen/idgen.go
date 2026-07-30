package idgen

import (
	"crypto/rand"
	"fmt"
)

// NewV4 generates a random UUID v4 string. Used for outbox EventID/dedup
// keys where Postgres row defaults (gen_random_uuid()) don't apply — the ID
// must exist in Go before the row is written.
func NewV4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
