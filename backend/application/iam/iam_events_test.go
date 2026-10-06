package iam_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appiam "github.com/adirkuhn/commerceos/application/iam"
	"github.com/adirkuhn/commerceos/application/iam/mocks"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func groupRepo(existing *iamDomain.Group) *mocks.GroupRepo {
	return &mocks.GroupRepo{
		CreateFn: func(_ context.Context, g *iamDomain.Group) error { g.ID = "group-1"; return nil },
		FindByIDFn: func(context.Context, string) (*iamDomain.Group, error) {
			if existing == nil {
				return nil, iamDomain.ErrGroupNotFound
			}
			cp := *existing
			return &cp, nil
		},
		FindByNameFn: func(context.Context, string) (*iamDomain.Group, error) { return nil, iamDomain.ErrGroupNotFound },
		UpdateFn:     func(context.Context, *iamDomain.Group) error { return nil },
		DeleteFn:     func(context.Context, string) error { return nil },
	}
}

func memberRepo(isMember bool) *mocks.UserGroupRepo {
	return &mocks.UserGroupRepo{
		IsMemberFn:     func(context.Context, string, string) (bool, error) { return isMember, nil },
		AddMemberFn:    func(context.Context, string, string) error { return nil },
		RemoveMemberFn: func(context.Context, string, string) error { return nil },
	}
}

func permissionRepo(rules ...iamDomain.Rule) *mocks.GroupPermissionRepo {
	return &mocks.GroupPermissionRepo{
		ListByGroupFn: func(context.Context, string) ([]iamDomain.Rule, error) { return rules, nil },
		AssignFn:      func(context.Context, string, iamDomain.Rule) error { return nil },
		RevokeFn:      func(context.Context, string, iamDomain.Rule) error { return nil },
	}
}

var existingGroup = &iamDomain.Group{ID: "group-1", Name: "Warehouse", Description: "Pickers"}

func TestCreateGroup_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appiam.NewCreateGroupUseCase(newMockGormDB(t), groupRepo(nil), outbox)

	_, err := uc.Execute(context.Background(), appiam.CreateGroupInput{Name: "Warehouse", Description: "Pickers"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "iam.group.created", e.Subject)
	assert.Equal(t, "group-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Warehouse", "description": "Pickers"}, e.Data())
}

func TestUpdateGroup_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appiam.NewUpdateGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), outbox)

	_, err := uc.Execute(context.Background(), appiam.UpdateGroupInput{ID: "group-1", Name: "Warehouse", Description: "Pickers + packers"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "iam.group.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"description": {"Pickers", "Pickers + packers"}}, e.Changes())
}

func TestDeleteGroup_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appiam.NewDeleteGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), outbox)

	require.NoError(t, uc.Execute(context.Background(), "group-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "iam.group.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Warehouse"}, e.Data())
}

func TestGroupMembership_RecordsAddedAndRemoved(t *testing.T) {
	added := &eventrecordtest.Outbox{}
	require.NoError(t, appiam.NewAddUserToGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), memberRepo(false), added).
		Execute(context.Background(), appiam.AddUserToGroupInput{GroupID: "group-1", UserID: "user-1"}))
	e := onlyEvent(t, added)
	assert.Equal(t, "iam.group_member.added", e.Subject)
	assert.Equal(t, "group-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"user_id": "user-1"}, e.Data())

	removed := &eventrecordtest.Outbox{}
	require.NoError(t, appiam.NewRemoveUserFromGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), memberRepo(true), removed).
		Execute(context.Background(), appiam.RemoveUserFromGroupInput{GroupID: "group-1", UserID: "user-1"}))
	assert.Equal(t, "iam.group_member.removed", onlyEvent(t, removed).Subject)
}

func TestGroupPermissions_RecordsAssignedAndRevoked(t *testing.T) {
	assigned := &eventrecordtest.Outbox{}
	require.NoError(t, appiam.NewAssignPermissionToGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), permissionRepo(), assigned).
		Execute(context.Background(), appiam.AssignPermissionInput{GroupID: "group-1", Rule: "pms.product.create"}))
	e := onlyEvent(t, assigned)
	assert.Equal(t, "iam.group_permission.assigned", e.Subject)
	assert.Equal(t, map[string]any{"rule": "pms.product.create"}, e.Data())

	revoked := &eventrecordtest.Outbox{}
	require.NoError(t, appiam.NewRevokePermissionFromGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), permissionRepo("pms.product.create"), revoked).
		Execute(context.Background(), appiam.RevokePermissionInput{GroupID: "group-1", Rule: "pms.product.create"}))
	assert.Equal(t, "iam.group_permission.revoked", onlyEvent(t, revoked).Subject)
}

// Granting a rule the group already has changes nothing.
func TestAssignPermission_AlreadyAssignedRecordsNothing(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	require.NoError(t, appiam.NewAssignPermissionToGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), permissionRepo("pms.product.create"), outbox).
		Execute(context.Background(), appiam.AssignPermissionInput{GroupID: "group-1", Rule: "pms.product.create"}))
	assert.Empty(t, outbox.Events)
}

func TestGeneratePairingCode_RecordsCreatedWithoutSecrets(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubScannerPairingCodeRepo{}
	uc := appiam.NewGeneratePairingCodeUseCase(newMockGormDB(t), repo, outbox)

	p, err := uc.Execute(context.Background(), appiam.GeneratePairingCodeInput{UserID: "user-1", Role: "customer_owner"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "iam.scanner_pairing_code.created", e.Subject)
	assert.Equal(t, "pairing-1", e.Payload["entity_id"])
	assert.Equal(t, "user-1", e.Data()["user_id"])
	assert.Equal(t, "customer_owner", e.Data()["role"])
	assert.NotContains(t, e.Data(), "code")
	assert.NotContains(t, e.Data(), "token")
	assert.NotContains(t, e.Data(), p.Code)
}

// The scanner claims unauthenticated: the event carries the pairing's
// tenant and acts as the pairing user.
func TestClaimPairingCode_RecordsClaimedForThePairingTenantAndUser(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubScannerPairingCodeRepo{findPendingResult: &iamDomain.ScannerPairingCode{
		ID: "pairing-1", TenantID: "tenant-1", UserID: "user-1", Role: "customer_owner", Status: iamDomain.ScannerPairingCodeStatusPending,
	}}
	uc := appiam.NewClaimPairingCodeUseCase(newMockGormDB(t), repo, []byte("secret"), outbox)

	_, err := uc.Execute(context.Background(), "123456")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "iam.scanner_pairing_code.claimed", e.Subject)
	assert.Equal(t, "tenant-1", e.Payload["tenant_id"])
	assert.Equal(t, map[string]any{"type": "user", "user_id": "user-1", "role": "customer_owner"}, e.Payload["actor"])
	assert.Equal(t, map[string]any{"user_id": "user-1", "role": "customer_owner"}, e.Data())
}

func TestIAMUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()

	_, err := appiam.NewCreateGroupUseCase(newMockGormDB(t), groupRepo(nil), failing).Execute(ctx, appiam.CreateGroupInput{Name: "A"})
	assert.ErrorIs(t, err, boom, "create")
	_, err = appiam.NewUpdateGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), failing).Execute(ctx, appiam.UpdateGroupInput{ID: "group-1", Name: "Warehouse"})
	assert.ErrorIs(t, err, boom, "update")
	assert.ErrorIs(t, appiam.NewDeleteGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), failing).Execute(ctx, "group-1"), boom, "delete")
	assert.ErrorIs(t, appiam.NewAddUserToGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), memberRepo(false), failing).
		Execute(ctx, appiam.AddUserToGroupInput{GroupID: "group-1", UserID: "u"}), boom, "add member")
	assert.ErrorIs(t, appiam.NewRemoveUserFromGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), memberRepo(true), failing).
		Execute(ctx, appiam.RemoveUserFromGroupInput{GroupID: "group-1", UserID: "u"}), boom, "remove member")
	assert.ErrorIs(t, appiam.NewAssignPermissionToGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), permissionRepo(), failing).
		Execute(ctx, appiam.AssignPermissionInput{GroupID: "group-1", Rule: "pms.product.create"}), boom, "assign")
	assert.ErrorIs(t, appiam.NewRevokePermissionFromGroupUseCase(newMockGormDB(t), groupRepo(existingGroup), permissionRepo("pms.product.create"), failing).
		Execute(ctx, appiam.RevokePermissionInput{GroupID: "group-1", Rule: "pms.product.create"}), boom, "revoke")
	_, err = appiam.NewGeneratePairingCodeUseCase(newMockGormDB(t), &stubScannerPairingCodeRepo{}, failing).Execute(ctx, appiam.GeneratePairingCodeInput{UserID: "u"})
	assert.ErrorIs(t, err, boom, "generate")
}
