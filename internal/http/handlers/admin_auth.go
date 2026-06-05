package handlers

import (
	"errors"
	"net/http"
	"strings"

	"4vpx/internal/http/middleware"
	"4vpx/internal/service"
)

func (a *App) LoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.SessionManager.AdminID(r); ok {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
		return
	}
	a.render(w, r, TemplateData{
		Title:           "Admin Login",
		ContentTemplate: "admin_login_content",
		Error:           a.errText(r),
		Flash:           a.flash(r),
		CurrentPath:     "/login",
	})
}

func (a *App) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, "/login", "error", "表单解析失败")
		return
	}

	admin, err := a.AdminService.Authenticate(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			redirectWithMessage(w, r, "/login", "error", "账号或密码错误")
			return
		}
		redirectWithMessage(w, r, "/login", "error", err.Error())
		return
	}
	if err := a.SessionManager.Create(r.Context(), w, admin.ID); err != nil {
		redirectWithMessage(w, r, "/login", "error", "创建登录会话失败")
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (a *App) Logout(w http.ResponseWriter, r *http.Request) {
	a.SessionManager.Destroy(w, r)
	redirectWithMessage(w, r, "/login", "flash", "已退出登录")
}

func (a *App) ChangePassword(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.CurrentAdminID(r)
	if adminID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", "表单解析失败")
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	if strings.TrimSpace(newPassword) == "" {
		redirectWithMessage(w, r, "/admin/system", "error", "新密码不能为空")
		return
	}

	if err := a.AdminService.ChangePassword(r.Context(), adminID, currentPassword, newPassword); err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", err.Error())
		return
	}
	redirectWithMessage(w, r, "/admin/system", "flash", "管理员密码已更新")
}
