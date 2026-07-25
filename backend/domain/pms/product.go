package pms

import "time"

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusArchived ProductStatus = "archived"
)

type Product struct {
	ID          string
	TenantID    string
	CategoryID  *string
	Name        string
	Slug        string
	Description string
	Status      ProductStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
