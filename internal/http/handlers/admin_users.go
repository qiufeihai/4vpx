package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"4vpx/internal/service"
)

func (a *App) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.UserService.List(r.Context(), time.Now().UTC())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.render(w, r, TemplateData{
		Title:           "Users",
		ContentTemplate: "admin_users_content",
		CurrentPath:     "/admin/users",
		Users:           users,
		Flash:           a.flash(r),
		Error:           a.errText(r),
	})
}

func (a *App) CreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, "/admin/users", "error", "表单解析失败")
		return
	}
	deviceSlots, err := parseInt(r.FormValue("device_slots"))
	if err != nil {
		redirectWithMessage(w, r, "/admin/users", "error", "设备位数量格式不正确")
		return
	}
	expiresAt, err := parseDateTime(r.FormValue("expires_at"))
	if err != nil {
		redirectWithMessage(w, r, "/admin/users", "error", "到期时间格式不正确")
		return
	}

	user, err := a.UserService.Create(r.Context(), service.CreateUserInput{
		Name:        r.FormValue("name"),
		Notes:       r.FormValue("notes"),
		Enabled:     r.FormValue("enabled") != "0",
		ExpiresAt:   expiresAt,
		DeviceSlots: deviceSlots,
	})
	if err != nil {
		redirectWithMessage(w, r, "/admin/users", "error", err.Error())
		return
	}

	if err := a.publishMutation(r, "用户创建成功"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", user.ID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", user.ID), "flash", "用户创建成功")
}

func (a *App) ShowUserDetail(w http.ResponseWriter, r *http.Request, userID int64) {
	view, err := a.PortalService.GetByUserID(r.Context(), userID, time.Now().UTC())
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	accessURL := strings.TrimRight(a.BaseURL, "/") + "/u/" + view.User.AccessToken
	a.render(w, r, TemplateData{
		Title:           "User Detail",
		ContentTemplate: "admin_user_detail_content",
		CurrentPath:     "/admin/users",
		User:            view.User,
		Devices:         view.Devices,
		Renewals:        view.Renewal,
		System:          view.System,
		AccessURL:       accessURL,
		Flash:           a.flash(r),
		Error:           a.errText(r),
	})
}

func (a *App) UpdateUser(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "表单解析失败")
		return
	}
	expiresAt, err := parseDateTime(r.FormValue("expires_at"))
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "到期时间格式不正确")
		return
	}
	_, err = a.UserService.Update(r.Context(), userID, service.UpdateUserInput{
		Name:      r.FormValue("name"),
		Notes:     r.FormValue("notes"),
		Enabled:   r.FormValue("enabled") != "0",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "用户更新成功"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "用户更新成功")
}

func (a *App) ToggleUser(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "表单解析失败")
		return
	}
	enabled := r.FormValue("enabled") == "1"
	_, err := a.UserService.SetEnabled(r.Context(), userID, enabled)
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "用户状态已更新"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "用户状态已更新")
}

func (a *App) DeleteUser(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := a.UserService.Delete(r.Context(), userID); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "用户已删除"); err != nil {
		redirectWithMessage(w, r, "/admin/users", "error", err.Error())
		return
	}
	redirectWithMessage(w, r, "/admin/users", "flash", "用户已删除")
}

func (a *App) publishMutation(r *http.Request, successMessage string) error {
	_, err := a.PublishService.Publish(r.Context())
	if err != nil {
		return fmt.Errorf("%s，但发布 Xray 配置失败: %w", successMessage, err)
	}
	return nil
}
