package middleware

import (
	"context"
	"net/http"
	"time"

	"4vpx/internal/domain"
	"4vpx/internal/security"
	"4vpx/internal/storage/sqlite"
)

type contextKey string

const adminIDContextKey contextKey = "admin_id"

type SessionManager struct {
	store      *sqlite.SessionRepository
	cookieName string
	secure     bool
	maxAge     int
}

func NewSessionManager(store *sqlite.SessionRepository, cookieName string, secure bool) *SessionManager {
	return &SessionManager{
		store:      store,
		cookieName: cookieName,
		secure:     secure,
		maxAge:     int((7 * 24 * time.Hour).Seconds()),
	}
}

func (s *SessionManager) Create(ctx context.Context, w http.ResponseWriter, adminID int64) error {
	token, err := security.NewSessionToken()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := s.store.Create(ctx, domain.AdminSession{
		Token:     token,
		AdminID:   adminID,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   s.maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
	return nil
}

func (s *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(s.cookieName)
	if err == nil && cookie.Value != "" {
		_ = s.store.DeleteByToken(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.secure,
	})
}

func (s *SessionManager) AdminID(r *http.Request) (int64, bool) {
	cookie, err := r.Cookie(s.cookieName)
	if err != nil || cookie.Value == "" {
		return 0, false
	}
	session, err := s.store.GetByToken(r.Context(), cookie.Value)
	if err != nil {
		return 0, false
	}
	return session.AdminID, true
}

func (s *SessionManager) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminID, ok := s.AdminID(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), adminIDContextKey, adminID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentAdminID(r *http.Request) int64 {
	adminID, _ := r.Context().Value(adminIDContextKey).(int64)
	return adminID
}
