package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"4vpx/internal/backup"
	"4vpx/internal/domain"
	"4vpx/internal/http/middleware"
	"4vpx/internal/service"
)

type App struct {
	Templates      *template.Template
	SessionManager *middleware.SessionManager
	CSRFManager    *middleware.CSRFManager
	AdminService   *service.AdminService
	UserService    *service.UserService
	DeviceService  *service.DeviceService
	RenewalService *service.RenewalService
	SystemService  *service.SystemService
	PublishService *service.PublishService
	PortalService  *service.UserPortalService
	Exporter       *backup.Exporter
	Importer       *backup.Importer
	BaseURL        string
}

type TemplateData struct {
	Title           string
	ContentTemplate string
	Flash           string
	Error           string
	CurrentPath     string
	AdminLoggedIn   bool
	BaseURL         string
	Users           []domain.User
	User            domain.User
	Devices         []service.PortalDevice
	Renewals        []domain.RenewalRecord
	System          domain.SystemConfig
	AccessURL       string
	PublishResult   string
	CSRFToken       string
}

func (a *App) render(w http.ResponseWriter, r *http.Request, data TemplateData) {
	data.AdminLoggedIn = middleware.CurrentAdminID(r) != 0
	data.BaseURL = strings.TrimRight(a.BaseURL, "/")
	token, err := a.CSRFManager.EnsureToken(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.CSRFToken = token
	if data.ContentTemplate == "" {
		http.Error(w, "missing template", http.StatusInternalServerError)
		return
	}
	if err := a.Templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) flash(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("flash"))
}

func (a *App) errText(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("error"))
}

func redirectWithMessage(w http.ResponseWriter, r *http.Request, path, key, message string) {
	target := path
	if strings.TrimSpace(message) != "" {
		target = path + "?" + key + "=" + url.QueryEscape(message)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func parseInt64(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

func parseInt(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}

func ParseUserID(value string) (int64, error) {
	return parseInt64(value)
}

func ParseSlotIndex(value string) (int, error) {
	return parseInt(value)
}

func parseDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid datetime: %s", value)
}
