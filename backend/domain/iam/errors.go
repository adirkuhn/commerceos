package iam

import "errors"

var (
	ErrGroupNotFound         = errors.New("group not found")
	ErrGroupNameTaken        = errors.New("group name already exists")
	ErrPermissionInvalid     = errors.New("invalid permission rule: expected {system}.{resource}.{action}")
	ErrAlreadyMember         = errors.New("user is already a member of this group")
	ErrNotMember             = errors.New("user is not a member of this group")
	ErrPermissionNotAssigned = errors.New("permission rule not assigned to this group")
)
