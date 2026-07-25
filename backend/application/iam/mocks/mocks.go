package mocks

import (
	"context"

	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
)

// GroupRepo mock

type GroupRepo struct {
	CreateFn     func(ctx context.Context, g *iamDomain.Group) error
	FindByIDFn   func(ctx context.Context, id string) (*iamDomain.Group, error)
	FindByNameFn func(ctx context.Context, name string) (*iamDomain.Group, error)
	ListFn       func(ctx context.Context) ([]iamDomain.Group, error)
	UpdateFn     func(ctx context.Context, g *iamDomain.Group) error
	DeleteFn     func(ctx context.Context, id string) error
}

func (m *GroupRepo) Create(ctx context.Context, g *iamDomain.Group) error {
	return m.CreateFn(ctx, g)
}
func (m *GroupRepo) FindByID(ctx context.Context, id string) (*iamDomain.Group, error) {
	return m.FindByIDFn(ctx, id)
}
func (m *GroupRepo) FindByName(ctx context.Context, name string) (*iamDomain.Group, error) {
	return m.FindByNameFn(ctx, name)
}
func (m *GroupRepo) List(ctx context.Context) ([]iamDomain.Group, error) {
	return m.ListFn(ctx)
}
func (m *GroupRepo) Update(ctx context.Context, g *iamDomain.Group) error {
	return m.UpdateFn(ctx, g)
}
func (m *GroupRepo) Delete(ctx context.Context, id string) error {
	return m.DeleteFn(ctx, id)
}

// UserGroupRepo mock

type UserGroupRepo struct {
	AddMemberFn    func(ctx context.Context, groupID, userID string) error
	RemoveMemberFn func(ctx context.Context, groupID, userID string) error
	ListMembersFn  func(ctx context.Context, groupID string) ([]string, error)
	IsMemberFn     func(ctx context.Context, groupID, userID string) (bool, error)
}

func (m *UserGroupRepo) AddMember(ctx context.Context, groupID, userID string) error {
	return m.AddMemberFn(ctx, groupID, userID)
}
func (m *UserGroupRepo) RemoveMember(ctx context.Context, groupID, userID string) error {
	return m.RemoveMemberFn(ctx, groupID, userID)
}
func (m *UserGroupRepo) ListMembers(ctx context.Context, groupID string) ([]string, error) {
	return m.ListMembersFn(ctx, groupID)
}
func (m *UserGroupRepo) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	return m.IsMemberFn(ctx, groupID, userID)
}

// GroupPermissionRepo mock

type GroupPermissionRepo struct {
	AssignFn        func(ctx context.Context, groupID string, rule iamDomain.Rule) error
	RevokeFn        func(ctx context.Context, groupID string, rule iamDomain.Rule) error
	ListByGroupFn   func(ctx context.Context, groupID string) ([]iamDomain.Rule, error)
	ListByUserFn    func(ctx context.Context, userID string) ([]iamDomain.Rule, error)
	HasPermissionFn func(ctx context.Context, userID string, rule iamDomain.Rule) (bool, error)
}

func (m *GroupPermissionRepo) Assign(ctx context.Context, groupID string, rule iamDomain.Rule) error {
	return m.AssignFn(ctx, groupID, rule)
}
func (m *GroupPermissionRepo) Revoke(ctx context.Context, groupID string, rule iamDomain.Rule) error {
	return m.RevokeFn(ctx, groupID, rule)
}
func (m *GroupPermissionRepo) ListByGroup(ctx context.Context, groupID string) ([]iamDomain.Rule, error) {
	return m.ListByGroupFn(ctx, groupID)
}
func (m *GroupPermissionRepo) ListByUser(ctx context.Context, userID string) ([]iamDomain.Rule, error) {
	return m.ListByUserFn(ctx, userID)
}
func (m *GroupPermissionRepo) HasPermission(ctx context.Context, userID string, rule iamDomain.Rule) (bool, error) {
	return m.HasPermissionFn(ctx, userID, rule)
}
