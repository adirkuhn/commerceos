package events

// Theme events carry ids only, same minimal-payload convention as
// StorefrontOrderPlaced: a consumer re-reads current state.

type ThemeCreated struct {
	Base
	TenantID       string `json:"tenant_id"`
	ThemeID        string `json:"theme_id"`
	SalesChannelID string `json:"sales_channel_id"`
}

func (ThemeCreated) EventName() string { return "storefront.theme.created" }

// ThemePublished fires when a draft is sealed and goes live -- the signal
// for anything caching the channel's live storefront to refresh.
type ThemePublished struct {
	Base
	TenantID       string `json:"tenant_id"`
	ThemeID        string `json:"theme_id"`
	SalesChannelID string `json:"sales_channel_id"`
	VersionID      string `json:"version_id"`
	VersionNumber  int    `json:"version_number"`
}

func (ThemePublished) EventName() string { return "storefront.theme.published" }

type ThemeRolledBack struct {
	Base
	TenantID       string `json:"tenant_id"`
	ThemeID        string `json:"theme_id"`
	SalesChannelID string `json:"sales_channel_id"`
	VersionID      string `json:"version_id"`
	VersionNumber  int    `json:"version_number"`
}

func (ThemeRolledBack) EventName() string { return "storefront.theme.rolled_back" }

type ThemeDeleted struct {
	Base
	TenantID       string `json:"tenant_id"`
	ThemeID        string `json:"theme_id"`
	SalesChannelID string `json:"sales_channel_id"`
}

func (ThemeDeleted) EventName() string { return "storefront.theme.deleted" }
