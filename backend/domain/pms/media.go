package pms

import "time"

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

type Media struct {
	ID        string
	TenantID  string
	ProductID *string
	VariantID *string
	URL       string
	Type      MediaType
	Position  int
	AltText   string
	CreatedAt time.Time
}
