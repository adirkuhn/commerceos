package pms

import "errors"

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrCategoryNotFound  = errors.New("category not found")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrSKUNotFound       = errors.New("sku not found")
	ErrDuplicateSlug     = errors.New("slug already exists")
	ErrDuplicateSKUCode  = errors.New("sku code already exists")
)
