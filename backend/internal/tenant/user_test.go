package tenant_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/internal/tenant"
)

func TestUserIDFromContext(t *testing.T) {
	ctx := tenant.WithUserID(context.Background(), "user-1")
	assert.Equal(t, "user-1", tenant.UserIDFromContext(ctx))
	assert.Equal(t, "", tenant.UserIDFromContext(context.Background()), "scheduler / consumer: no user")
}
