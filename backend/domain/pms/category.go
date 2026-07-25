package pms

import "time"

type Category struct {
	ID        string
	TenantID  string
	ParentID  *string
	Name      string
	Slug      string
	Position  int
	CreatedAt time.Time
	UpdatedAt time.Time
}
