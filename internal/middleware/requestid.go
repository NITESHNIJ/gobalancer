package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey string

const ctxRequestID ctxKey = "request_id"

// RequestID injects a UUID v4 as X-Request-ID on every request.
// If the client already sent an X-Request-ID, it is preserved (pass-through).
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		r = r.WithContext(context.WithValue(r.Context(), ctxRequestID, id))
		r.Header.Set("X-Request-ID", id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// RequestIDFromContext retrieves the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxRequestID).(string)
	return id
}
