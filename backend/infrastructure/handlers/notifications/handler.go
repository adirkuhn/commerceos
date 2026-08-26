package notifications

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	gennotifications "github.com/adirkuhn/commerceos/api/notifications"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type listNotificationsUC interface {
	Execute(ctx context.Context) ([]notificationsDomain.Notification, error)
}
type markNotificationReadUC interface {
	Execute(ctx context.Context, id string) error
}
type markAllNotificationsReadUC interface {
	Execute(ctx context.Context) error
}

type Handler struct {
	listNotifications        listNotificationsUC
	markNotificationRead     markNotificationReadUC
	markAllNotificationsRead markAllNotificationsReadUC
}

func NewHandler(
	listNotifications listNotificationsUC,
	markNotificationRead markNotificationReadUC,
	markAllNotificationsRead markAllNotificationsReadUC,
) *Handler {
	return &Handler{
		listNotifications: listNotifications, markNotificationRead: markNotificationRead,
		markAllNotificationsRead: markAllNotificationsRead,
	}
}

var _ gennotifications.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	repo := repositories.NewNotificationRepo(db)
	h := NewHandler(
		appnotifications.NewListNotificationsUseCase(db, repo),
		appnotifications.NewMarkNotificationReadUseCase(db, repo),
		appnotifications.NewMarkAllNotificationsReadUseCase(db, repo),
	)
	gennotifications.RegisterHandlers(rg, h)
}

func (h *Handler) ListNotifications(c *gin.Context) {
	list, err := h.listNotifications.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]gennotifications.Notification, len(list))
	for i, n := range list {
		resp[i] = toGenNotification(n)
	}
	c.JSON(http.StatusOK, gennotifications.NotificationListResponse{Data: &resp})
}

func (h *Handler) MarkNotificationRead(c *gin.Context, id gennotifications.Id) {
	err := h.markNotificationRead.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gennotifications.EmptyResponse{})
}

func (h *Handler) MarkAllNotificationsRead(c *gin.Context) {
	err := h.markAllNotificationsRead.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, gennotifications.EmptyResponse{})
}

func toGenNotification(n notificationsDomain.Notification) gennotifications.Notification {
	link := n.Link
	return gennotifications.Notification{
		Id: n.ID, Type: string(n.Type), Title: n.Title, Message: n.Message, Link: &link,
		Read: n.Read, CreatedAt: n.CreatedAt,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	log.Printf("notifications: unhandled error: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	return err
}
