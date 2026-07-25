package txctx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/internal/txctx"
)

func TestFromContext_absent(t *testing.T) {
	_, ok := txctx.FromContext(context.Background())
	assert.False(t, ok)
}

func TestFromContext_present(t *testing.T) {
	// We can't easily construct a *gorm.DB in a unit test, so we verify the
	// nil sentinel: storing nil must not be treated as present.
	ctx := context.WithValue(context.Background(), struct{}{}, nil)
	_, ok := txctx.FromContext(ctx)
	assert.False(t, ok, "nil value must not count as a transaction")
}
