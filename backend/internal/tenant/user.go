package tenant

import "context"

const userIDKey contextKey = "user_id"

// WithUserID records who is acting (the lm2 JWT user_id), set by
// middleware.JWT -- domain events name it as their actor. Absent for
// scheduler jobs and event consumers, which act as the system.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}
