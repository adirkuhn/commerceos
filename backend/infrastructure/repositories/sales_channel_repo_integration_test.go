//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestSalesChannelRepo_Integration_CreateAcceptsBolType is a regression
// test for a real prod bug: the sales_channels_type_check CHECK constraint
// (migration 037) was never widened to 'bol' when Bol integration support
// was added, so a channel with Type=ChannelTypeBol -- already valid at the
// Go domain level (ChannelType.Valid()) and at the OpenAPI request layer
// (both fixed in earlier commits) -- still failed at INSERT with a raw
// constraint violation. This can only be caught against a real Postgres,
// not sqlmock, which never enforces CHECK constraints.
func TestSalesChannelRepo_Integration_CreateAcceptsBolType(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	repo := repositories.NewSalesChannelRepo(db)

	var created channelsDomain.SalesChannel
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created = channelsDomain.SalesChannel{
			TenantID: tenant.IDFromContext(ctx), Name: "BOL", Type: channelsDomain.ChannelTypeBol,
			EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true, CreatedAt: now, UpdatedAt: now,
		}
		return repo.Create(ctx, &created)
	})

	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var found *channelsDomain.SalesChannel
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = repo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, channelsDomain.ChannelTypeBol, found.Type)
}
