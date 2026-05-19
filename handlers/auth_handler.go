package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type AuthHandler struct {
	AuthSvc *services.AuthService
}

func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{AuthSvc: authSvc}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	hasPassword, _ := h.AuthSvc.HasPassword()
	if !hasPassword {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	if err := h.AuthSvc.VerifyPassword(req.Password); err != nil {
		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "waypoint_session",
		Value:    "authenticated",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "waypoint_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	hasPassword, _ := h.AuthSvc.HasPassword()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"has_password": hasPassword})
}

func (h *AuthHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 4 {
		http.Error(w, "password must be at least 4 characters", http.StatusBadRequest)
		return
	}

	if err := h.AuthSvc.SetPassword(req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "waypoint_session",
		Value:    "authenticated",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hasPassword, _ := authSvc.HasPassword()
			if !hasPassword {
				next.ServeHTTP(w, r)
				return
			}

			if r.URL.Path == "/api/v1/auth/login" ||
				r.URL.Path == "/api/v1/auth/status" ||
				r.URL.Path == "/api/v1/auth/set-password" {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie("waypoint_session")
			if err != nil || cookie.Value != "authenticated" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
