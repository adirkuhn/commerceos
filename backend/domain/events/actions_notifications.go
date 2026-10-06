package events

// Notification actions (backend/application/notifications).
var (
	NotificationCreated = defineAction("notifications.notification.created",
		"An in-app notification (bell) was created. data: type, title, link.")
	NotificationRead = defineAction("notifications.notification.read",
		"A notification was marked as read.")
	NotificationsAllRead = defineAction("notifications.notification.all_read",
		"Every notification of the tenant was marked as read. entity_id is the tenant id.")
)
