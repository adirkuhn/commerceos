package iam

import "strings"

// Rule format: {system}.{resource}.{action} — e.g. pms.product.create, oms.order.read
type Rule string

func (r Rule) Validate() error {
	parts := strings.Split(string(r), ".")
	if len(parts) != 3 {
		return ErrPermissionInvalid
	}
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			return ErrPermissionInvalid
		}
	}
	return nil
}

func (r Rule) String() string { return string(r) }
