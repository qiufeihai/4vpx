package middleware

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"4vpx/internal/security"
)

var ErrInvalidCSRFToken = errors.New("invalid csrf token")

type CSRFManager struct {
	cookieName string
	secure     bool
}

func NewCSRFManager(secure bool) *CSRFManager {
	return &CSRFManager{
		cookieName: "csrf_token",
		secure:     secure,
	}
}

func (m *CSRFManager) EnsureToken(w http.ResponseWriter, r *http.Request) (string, error) {
	if cookie, err := r.Cookie(m.cookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value, nil
	}

	token, err := security.NewCSRFToken()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   m.secure,
	})
	return token, nil
}

func (m *CSRFManager) Verify(r *http.Request) error {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return ErrInvalidCSRFToken
	}
	formValue := strings.TrimSpace(r.FormValue("csrf_token"))
	if formValue == "" {
		return ErrInvalidCSRFToken
	}
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(formValue)) != 1 {
		return ErrInvalidCSRFToken
	}
	return nil
}

func (m *CSRFManager) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if err := m.Verify(r); err != nil {
			http.Error(w, "csrf token invalid", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
