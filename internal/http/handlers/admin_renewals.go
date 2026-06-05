package handlers

import (
	"fmt"
	"net/http"
	"time"
)

func (a *App) Renew7Days(w http.ResponseWriter, r *http.Request, userID int64) {
	a.renewDays(w, r, userID, 7)
}

func (a *App) Renew1Month(w http.ResponseWriter, r *http.Request, userID int64) {
	_, _, err := a.RenewalService.RenewMonth(r.Context(), userID, "admin", r.FormValue("notes"), time.Now().UTC())
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "续费成功"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "已续费 1 个月")
}

func (a *App) RenewCustom(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "表单解析失败")
		return
	}
	target, err := parseDateTime(r.FormValue("target_expires_at"))
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "目标到期时间格式不正确")
		return
	}
	_, _, err = a.RenewalService.ExtendTo(r.Context(), userID, target, "admin", r.FormValue("notes"), time.Now().UTC())
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "续费成功"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "自定义续费成功")
}

func (a *App) renewDays(w http.ResponseWriter, r *http.Request, userID int64, days int) {
	_, _, err := a.RenewalService.RenewDays(r.Context(), userID, days, "admin", r.FormValue("notes"), time.Now().UTC())
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "续费成功"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", fmt.Sprintf("已续费 %d 天", days))
}
