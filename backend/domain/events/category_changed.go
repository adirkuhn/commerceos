package events

// CategoryChanged is raised whenever a category's fields (name, parent,
// markup) change. Unlike ProductChanged, no consumer acts on this yet --
// each channel's own sync logic decides later whether/how a category
// rename should reach it (e.g. Woo categories are today only matched by
// slug and linked, never renamed from CommerceOS's side). The event only
// carries the category id, not a field snapshot, same reasoning as
// ProductChanged.
type CategoryChanged struct {
	Base
	TenantID   string `json:"tenant_id"`
	CategoryID string `json:"category_id"`
}

func (CategoryChanged) EventName() string { return "pms.category.changed" }
