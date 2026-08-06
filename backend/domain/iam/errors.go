package iam

import "errors"

var (
	ErrGroupNotFound         = errors.New("group not found")
	ErrGroupNameTaken        = errors.New("group name already exists")
	ErrPermissionInvalid     = errors.New("invalid permission rule: expected {system}.{resource}.{action}")
	ErrAlreadyMember         = errors.New("user is already a member of this group")
	ErrNotMember             = errors.New("user is not a member of this group")
	ErrPermissionNotAssigned = errors.New("permission rule not assigned to this group")

	ErrPairingCodeNotFound = errors.New("pairing code not found")
	// ErrPairingCodeInvalid covers both "no pending code with this text" and
	// "expired" -- ClaimPairingCodeUseCase deliberately doesn't distinguish
	// them to a caller with no tenant identity yet, same reasoning as auth
	// endpoints not revealing whether a username exists.
	ErrPairingCodeInvalid = errors.New("pairing code is invalid or expired")
)
