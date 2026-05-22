package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type AuthHandler struct {
	AuthSvc *services.AuthService
}

func NewAuthHandler(authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{AuthSvc: authSvc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.AuthSvc.Register(req.Username, req.Password)
	if err != nil {
		code := http.StatusInternalServerError
		if err == services.ErrUsernameTaken {
			code = http.StatusConflict
		} else if err.Error() == "username and password are required" {
			code = http.StatusBadRequest
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	token, _, err := h.AuthSvc.Login(req.Username, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "waypoint_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token, user, err := h.AuthSvc.Login(req.Username, req.Password)
	if err != nil {
		code := http.StatusUnauthorized
		if err == services.ErrInvalidCredentials {
			code = http.StatusUnauthorized
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "waypoint_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("waypoint_session")
	if err == nil {
		h.AuthSvc.Logout(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "waypoint_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	status := models.AuthStatus{Authenticated: false}

	cookie, err := r.Cookie("waypoint_session")
	if err == nil {
		user, err := h.AuthSvc.GetSessionUser(cookie.Value)
		if err == nil {
			status.Authenticated = true
			status.User = user
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *AuthHandler) HasUsers(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := h.AuthSvc.HasUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"has_users": hasUsers})
}

func AuthMiddleware(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("waypoint_session")
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			user, err := authSvc.GetSessionUser(cookie.Value)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}

			r = WithUser(r, user.ID, user.Username)
			next.ServeHTTP(w, r)
		})
	}
}
