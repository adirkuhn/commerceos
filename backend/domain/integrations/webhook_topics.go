package integrations

// WooWebhookTopic is one of WooCommerce's own fixed REST API webhook topic
// names (Woo docs: Webhooks -> Topics). Typed so call sites reference a
// named constant instead of a raw string -- WooClient.CreateWebhook still
// takes a plain string (WooCommerce itself is the source of truth and
// rejects an unknown one; this codebase doesn't re-validate), callers just
// have a canonical value to pass instead of retyping the literal.
type WooWebhookTopic string

const (
	WooTopicOrderCreated    WooWebhookTopic = "order.created"
	WooTopicOrderUpdated    WooWebhookTopic = "order.updated"
	WooTopicOrderDeleted    WooWebhookTopic = "order.deleted"
	WooTopicProductCreated  WooWebhookTopic = "product.created"
	WooTopicProductUpdated  WooWebhookTopic = "product.updated"
	WooTopicProductDeleted  WooWebhookTopic = "product.deleted"
	WooTopicCustomerCreated WooWebhookTopic = "customer.created"
	WooTopicCustomerUpdated WooWebhookTopic = "customer.updated"
	WooTopicCustomerDeleted WooWebhookTopic = "customer.deleted"
	WooTopicCouponCreated   WooWebhookTopic = "coupon.created"
	WooTopicCouponUpdated   WooWebhookTopic = "coupon.updated"
	WooTopicCouponDeleted   WooWebhookTopic = "coupon.deleted"
)

// AllWooWebhookTopics lists every WooWebhookTopic -- the fan-out rule
// form's Topic dropdown seed for provider "woocommerce" (see
// application/webhooks.ListAvailableWebhookTopicsUseCase).
var AllWooWebhookTopics = []WooWebhookTopic{
	WooTopicOrderCreated, WooTopicOrderUpdated, WooTopicOrderDeleted,
	WooTopicProductCreated, WooTopicProductUpdated, WooTopicProductDeleted,
	WooTopicCustomerCreated, WooTopicCustomerUpdated, WooTopicCustomerDeleted,
	WooTopicCouponCreated, WooTopicCouponUpdated, WooTopicCouponDeleted,
}

// MyParcelHookType is one of MyParcel's own three webhook hook types (see
// developer.myparcel.nl/api-reference/10.webhooks.html -- "we currently
// have three available hooks"). Same typed-constant convention as
// WooWebhookTopic; MyParcelClient.CreateWebhookSubscription still takes a
// plain string for the same reason.
type MyParcelHookType string

const (
	MyParcelHookShipmentStatusChange MyParcelHookType = "shipment_status_change"
	MyParcelHookShipmentLabelCreated MyParcelHookType = "shipment_label_created"
	MyParcelHookOrderStatusChange    MyParcelHookType = "order_status_change"
)

// AllMyParcelHookTypes lists every MyParcelHookType -- the Topic dropdown
// seed for provider "myparcel".
var AllMyParcelHookTypes = []MyParcelHookType{
	MyParcelHookShipmentStatusChange, MyParcelHookShipmentLabelCreated, MyParcelHookOrderStatusChange,
}

// KnownWebhookTopicsByProvider is every topic this codebase knows about
// ahead of time, grouped by provider -- the baseline catalog
// ListAvailableWebhookTopicsUseCase seeds its result with, so a brand-new
// tenant (zero received events, zero subscriptions) still gets a populated
// Topic dropdown instead of an empty one.
func KnownWebhookTopicsByProvider() map[Provider][]string {
	woo := make([]string, len(AllWooWebhookTopics))
	for i, t := range AllWooWebhookTopics {
		woo[i] = string(t)
	}
	myParcel := make([]string, len(AllMyParcelHookTypes))
	for i, t := range AllMyParcelHookTypes {
		myParcel[i] = string(t)
	}
	return map[Provider][]string{
		ProviderWooCommerce: woo,
		ProviderMyParcel:    myParcel,
	}
}
