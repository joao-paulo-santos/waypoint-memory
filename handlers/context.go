package handlers

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"
const usernameKey contextKey = "username"

func WithUser(r *http.Request, userID int64, username string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	return r.WithContext(ctx)
}

func getUserID(r *http.Request) int64 {
	v, _ := r.Context().Value(userIDKey).(int64)
	return v
}

func getUsername(r *http.Request) string {
	v, _ := r.Context().Value(usernameKey).(string)
	return v
}
