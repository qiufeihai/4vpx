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

func (a *App) UserDeviceMihomo(w http.ResponseWriter, r *http.Request, token string, slotIndex int) {
	view, err := a.PortalService.GetDeviceByTokenAndSlot(r.Context(), strings.TrimSpace(token), slotIndex, time.Now().UTC())
	if err != nil {
		http.Error(w, "设备不存在或订阅链接无效", http.StatusNotFound)
		return
	}

	fileName := sanitizeDownloadName(view.User.Name, view.Device.Slot.Label) + ".yaml"
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="`+fileName+`"`)
	_, _ = w.Write([]byte(view.Device.MihomoYAML))
}

func (a *App) UserDeviceVLESS(w http.ResponseWriter, r *http.Request, token string, slotIndex int) {
	view, err := a.PortalService.GetDeviceByTokenAndSlot(r.Context(), strings.TrimSpace(token), slotIndex, time.Now().UTC())
	if err != nil {
		http.Error(w, "设备不存在或订阅链接无效", http.StatusNotFound)
		return
	}

	fileName := sanitizeDownloadName(view.User.Name, view.Device.Slot.Label) + ".txt"
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `inline; filename="`+fileName+`"`)
	_, _ = w.Write([]byte(view.Device.VLESSURI + "\n"))
}

func sanitizeDownloadName(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", `"`, "", "'", "")
		cleaned = append(cleaned, replacer.Replace(part))
	}
	if len(cleaned) == 0 {
		return "4vpx-device"
	}
	return strings.Join(cleaned, "-")
}
