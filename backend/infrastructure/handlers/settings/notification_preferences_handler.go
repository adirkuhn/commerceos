package settings

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	gensettings "github.com/adirkuhn/commerceos/api/settings"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type getNotificationPreferencesUC interface {
	Execute(ctx context.Context) (*notificationsDomain.Preferences, error)
}
type updateNotificationPreferencesUC interface {
	Execute(ctx context.Context, in notificationsDomain.PreferencesInput) (*notificationsDomain.Preferences, error)
}

// NotificationPreferencesHandler serves /settings/notification-preferences
// (embedded in Handler, like CompanyProfileHandler).
type NotificationPreferencesHandler struct {
	get    getNotificationPreferencesUC
	update updateNotificationPreferencesUC
}

func NewNotificationPreferencesHandler(get getNotificationPreferencesUC, update updateNotificationPreferencesUC) *NotificationPreferencesHandler {
	return &NotificationPreferencesHandler{get: get, update: update}
}

func (h *NotificationPreferencesHandler) GetNotificationPreferences(c *gin.Context) {
	p, err := h.get.Execute(c.Request.Context())
	if err != nil {
		failNotificationPreferences(c, err)
		return
	}
	resp := toGenNotificationPreferences(p)
	c.JSON(http.StatusOK, gensettings.NotificationPreferencesResponse{Data: &resp})
}

func (h *NotificationPreferencesHandler) UpdateNotificationPreferences(c *gin.Context) {
	var req gensettings.UpdateNotificationPreferencesJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusBadRequest, "INVALID_REQUEST", err)
		return
	}
	in := notificationsDomain.PreferencesInput{
		Categories:      make(map[notificationsDomain.Category]notificationsDomain.ChannelPrefs, len(req.Categories)),
		EmailRecipients: req.EmailRecipients,
	}
	for _, cp := range req.Categories {
		in.Categories[notificationsDomain.Category(cp.Category)] = notificationsDomain.ChannelPrefs{InApp: cp.InApp, Email: cp.Email}
	}
	p, err := h.update.Execute(c.Request.Context(), in)
	if err != nil {
		failNotificationPreferences(c, err)
		return
	}
	resp := toGenNotificationPreferences(p)
	c.JSON(http.StatusOK, gensettings.NotificationPreferencesResponse{Data: &resp})
}

var notificationPreferencesErrorCodes = []struct {
	err  error
	code string
}{
	{notificationsDomain.ErrInvalidRecipient, "INVALID_RECIPIENT"},
	{notificationsDomain.ErrTooManyRecipients, "TOO_MANY_RECIPIENTS"},
	{notificationsDomain.ErrEmailRecipientsRequired, "EMAIL_RECIPIENTS_REQUIRED"},
	{notificationsDomain.ErrUnknownCategory, "UNKNOWN_CATEGORY"},
}

func failNotificationPreferences(c *gin.Context, err error) {
	for _, m := range notificationPreferencesErrorCodes {
		if errors.Is(err, m.err) {
			middleware.Fail(c, http.StatusUnprocessableEntity, m.code, m.err.Error())
			return
		}
	}
	log.Printf("settings: notification preferences: %v", err)
	middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
}

func toGenNotificationPreferences(p *notificationsDomain.Preferences) gensettings.NotificationPreferences {
	out := gensettings.NotificationPreferences{EmailRecipients: p.EmailRecipients}
	if out.EmailRecipients == nil {
		out.EmailRecipients = []string{}
	}
	for _, c := range notificationsDomain.Categories {
		prefs := p.For(c)
		out.Categories = append(out.Categories, gensettings.NotificationCategoryPreference{
			Category: gensettings.NotificationCategory(c), InApp: prefs.InApp, Email: prefs.Email,
		})
	}
	if !p.UpdatedAt.IsZero() {
		t := p.UpdatedAt
		out.UpdatedAt = &t
	}
	return out
}
