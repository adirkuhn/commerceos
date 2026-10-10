package iam

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrGroupNotFound         = errs.New("IAM_GROUP_NOT_FOUND", "group not found")
	ErrGroupNameTaken        = errs.New("IAM_GROUP_NAME_TAKEN", "group name already exists")
	ErrPermissionInvalid     = errs.New("IAM_PERMISSION_INVALID", "invalid permission rule: expected {system}.{resource}.{action}")
	ErrAlreadyMember         = errs.New("IAM_ALREADY_MEMBER", "user is already a member of this group")
	ErrNotMember             = errs.New("IAM_NOT_MEMBER", "user is not a member of this group")
	ErrPermissionNotAssigned = errs.New("IAM_PERMISSION_NOT_ASSIGNED", "permission rule not assigned to this group")

	ErrPairingCodeNotFound = errs.New("PAIRING_CODE_NOT_FOUND", "pairing code not found")
	// ErrPairingCodeInvalid covers both "no pending code with this text" and
	// "expired" -- ClaimPairingCodeUseCase deliberately doesn't distinguish
	// them to a caller with no tenant identity yet, same reasoning as auth
	// endpoints not revealing whether a username exists.
	ErrPairingCodeInvalid = errs.New("PAIRING_CODE_INVALID", "pairing code is invalid or expired")
)
