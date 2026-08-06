package iam

import (
	"context"
	"time"
)

type GroupRepository interface {
	Create(ctx context.Context, g *Group) error
	FindByID(ctx context.Context, id string) (*Group, error)
	FindByName(ctx context.Context, name string) (*Group, error)
	List(ctx context.Context) ([]Group, error)
	Update(ctx context.Context, g *Group) error
	Delete(ctx context.Context, id string) error
}

type UserGroupRepository interface {
	AddMember(ctx context.Context, groupID, userID string) error
	RemoveMember(ctx context.Context, groupID, userID string) error
	ListMembers(ctx context.Context, groupID string) ([]string, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
}

type GroupPermissionRepository interface {
	Assign(ctx context.Context, groupID string, rule Rule) error
	Revoke(ctx context.Context, groupID string, rule Rule) error
	ListByGroup(ctx context.Context, groupID string) ([]Rule, error)
	ListByUser(ctx context.Context, userID string) ([]Rule, error)
	HasPermission(ctx context.Context, userID string, rule Rule) (bool, error)
}

type ScannerPairingCodeRepository interface {
	Create(ctx context.Context, p *ScannerPairingCode) error
	FindByID(ctx context.Context, id string) (*ScannerPairingCode, error)
	// FindPendingByCode is the only cross-tenant lookup here -- called by
	// ClaimPairingCodeUseCase under tenant.WithSuperAdmin since the scanner
	// claiming a code has no tenant identity yet (same cross-tenant-read
	// pattern cmd/scheduler uses for polling). Returns nil (not an error)
	// when no pending, unexpired code matches.
	FindPendingByCode(ctx context.Context, code string) (*ScannerPairingCode, error)
	MarkClaimed(ctx context.Context, id string, claimedAt time.Time) error
}
