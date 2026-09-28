package events

// CollectionChanged fires when a collection is created, edited, has its
// manual products changed, or is deleted (Deleted set) -- the signal for
// anything caching collection pages to refresh.
type CollectionChanged struct {
	Base
	TenantID     string `json:"tenant_id"`
	CollectionID string `json:"collection_id"`
	Deleted      bool   `json:"deleted"`
}

func (CollectionChanged) EventName() string { return "collections.collection.changed" }
