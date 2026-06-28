package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/ysksm/cc-otel/internal/auth"
	"github.com/ysksm/cc-otel/internal/model"
)

// authPublicPaths are reachable without a session even when auth is enabled.
func isAuthPublic(path string) bool {
	switch path {
	case "/api/health", "/api/auth/login", "/api/auth/logout", "/api/auth/me":
		return true
	}
	return false
}

// authGate enforces a valid session on /api/* (except public paths) when auth is
// enabled. /v1/* (ingestion) and static assets are unaffected.
func (s *Server) authGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth.Enabled && strings.HasPrefix(r.URL.Path, "/api/") && !isAuthPublic(r.URL.Path) {
			if _, ok := s.sessionAccountID(r); !ok {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// sessionAccountID returns the account id from a valid session cookie.
func (s *Server) sessionAccountID(r *http.Request) (string, bool) {
	c, err := r.Cookie(auth.SessionCookie)
	if err != nil {
		return "", false
	}
	return auth.VerifySession(s.auth.Secret, c.Value)
}

func (s *Server) currentAccount(r *http.Request) (model.Account, bool) {
	id, ok := s.sessionAccountID(r)
	if !ok {
		return model.Account{}, false
	}
	a, err := s.store.GetAccountByID(id)
	if err != nil {
		return model.Account{}, false
	}
	return a, true
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled {
		writeError(w, http.StatusBadRequest, "authentication is not enabled")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	acc, err := s.store.GetAccountByEmail(body.Email)
	if err != nil || !auth.CheckPassword(acc.PasswordHash, body.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	exp := time.Now().Add(auth.SessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    auth.SignSession(s.auth.Secret, acc.ID, exp),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  exp,
		MaxAge:   int(auth.SessionTTL / time.Second),
	})
	writeJSON(w, http.StatusOK, map[string]any{"account": acc})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookie, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if !s.auth.Enabled {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "authenticated": true})
		return
	}
	acc, ok := s.currentAccount(r)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": true, "authenticated": true, "account": acc})
}

func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	accounts, err := s.store.ListAccounts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if accounts == nil {
		accounts = []model.Account{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": accounts})
}

func (s *Server) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(body.Email) == "" || len(body.Password) < 6 {
		writeError(w, http.StatusBadRequest, "email and a password of at least 6 characters are required")
		return
	}
	hash, err := auth.HashPassword(body.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	acc, err := s.store.CreateAccount(body.Email, hash, body.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not create account (email may already exist)")
		return
	}
	writeJSON(w, http.StatusOK, acc)
}

// requireAdmin ensures auth is enabled and the caller is an admin.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !s.auth.Enabled {
		writeError(w, http.StatusBadRequest, "authentication is not enabled")
		return false
	}
	acc, ok := s.currentAccount(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if acc.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin role required")
		return false
	}
	return true
}
