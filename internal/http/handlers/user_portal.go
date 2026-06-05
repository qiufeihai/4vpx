package handlers

import (
	"net/http"
	"strings"
	"time"
)

func (a *App) UserPortal(w http.ResponseWriter, r *http.Request, token string) {
	view, err := a.PortalService.GetByToken(r.Context(), strings.TrimSpace(token), time.Now().UTC())
	if err != nil {
		http.Error(w, "用户不存在或访问链接无效", http.StatusNotFound)
		return
	}
	a.render(w, r, TemplateData{
		Title:           "User Portal",
		ContentTemplate: "user_portal_content",
		User:            view.User,
		Devices:         view.Devices,
		Renewals:        view.Renewal,
		System:          view.System,
	})
}
