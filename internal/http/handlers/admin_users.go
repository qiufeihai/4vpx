package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"4vpx/internal/service"
)

func (a *App) ListUsers(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	filter := service.UserListFilter{
		Query:    strings.TrimSpace(r.URL.Query().Get("q")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Expiry:   strings.TrimSpace(r.URL.Query().Get("expiry")),
		Page:     parsePositiveIntOrDefault(r.URL.Query().Get("page"), 1),
		PageSize: parsePositiveIntOrDefault(r.URL.Query().Get("page_size"), 20),
	}
	page, err := a.UserService.ListFiltered(r.Context(), filter, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	meta := buildUserListMeta(filter, page)
	a.render(w, r, TemplateData{
		Title:           "Users",
		ContentTemplate: "admin_users_content",
		CurrentPath:     "/admin/users",
		Users:           page.Items,
		Flash:           a.flash(r),
		Error:           a.errText(r),
		UserFilters: UserListFilters{
			Query:  filter.Query,
			Status: filter.Status,
			Expiry: filter.Expiry,
		},
		UserListMeta:  meta,
		UserActionURL: userListActionURL(r),
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
	expiresAt, err := resolveCreateUserExpiry(r.FormValue("quick_expiry"), r.FormValue("expires_at"), time.Now().UTC())
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
	target := redirectTarget(r, fmt.Sprintf("/admin/users/%d", userID))
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, target, "error", "表单解析失败")
		return
	}
	enabled := r.FormValue("enabled") == "1"
	_, err := a.UserService.SetEnabled(r.Context(), userID, enabled)
	if err != nil {
		redirectWithMessage(w, r, target, "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "用户状态已更新"); err != nil {
		redirectWithMessage(w, r, target, "error", err.Error())
		return
	}
	redirectWithMessage(w, r, target, "flash", "用户状态已更新")
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

func parsePositiveIntOrDefault(value string, fallback int) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func userListActionURL(r *http.Request) string {
	if r.URL == nil {
		return "/admin/users"
	}
	target := r.URL.RequestURI()
	if strings.TrimSpace(target) == "" {
		return "/admin/users"
	}
	return target
}

func buildUserListMeta(filter service.UserListFilter, page service.UserListPage) UserListMeta {
	showingFrom := 0
	showingTo := 0
	if page.Total > 0 && len(page.Items) > 0 {
		showingFrom = (page.Page-1)*page.PageSize + 1
		showingTo = showingFrom + len(page.Items) - 1
	}

	meta := UserListMeta{
		Total:       page.Total,
		Page:        page.Page,
		PageSize:    page.PageSize,
		TotalPages:  page.TotalPages,
		ShowingFrom: showingFrom,
		ShowingTo:   showingTo,
		HasPrev:     page.Page > 1,
		HasNext:     page.Page < page.TotalPages,
	}
	if meta.HasPrev {
		meta.PrevURL = buildUserListPageURL(filter, page.PageSize, page.Page-1)
	}
	if meta.HasNext {
		meta.NextURL = buildUserListPageURL(filter, page.PageSize, page.Page+1)
	}

	for _, pageNo := range visibleUserListPages(page.Page, page.TotalPages) {
		meta.Links = append(meta.Links, PaginationLink{
			Label:   strconv.Itoa(pageNo),
			URL:     buildUserListPageURL(filter, page.PageSize, pageNo),
			Current: pageNo == page.Page,
		})
	}
	return meta
}

func buildUserListPageURL(filter service.UserListFilter, pageSize int, page int) string {
	values := url.Values{}
	if strings.TrimSpace(filter.Query) != "" {
		values.Set("q", strings.TrimSpace(filter.Query))
	}
	if strings.TrimSpace(filter.Status) != "" {
		values.Set("status", strings.TrimSpace(filter.Status))
	}
	if strings.TrimSpace(filter.Expiry) != "" {
		values.Set("expiry", strings.TrimSpace(filter.Expiry))
	}
	if pageSize > 0 && pageSize != 20 {
		values.Set("page_size", strconv.Itoa(pageSize))
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/admin/users"
	}
	return "/admin/users?" + encoded
}

func visibleUserListPages(current int, total int) []int {
	if total <= 0 {
		return []int{1}
	}
	start := current - 2
	if start < 1 {
		start = 1
	}
	end := start + 4
	if end > total {
		end = total
	}
	if end-start < 4 {
		start = end - 4
		if start < 1 {
			start = 1
		}
	}
	pages := make([]int, 0, end-start+1)
	for page := start; page <= end; page++ {
		pages = append(pages, page)
	}
	return pages
}

func resolveCreateUserExpiry(quickExpiry string, customValue string, now time.Time) (time.Time, error) {
	switch strings.TrimSpace(quickExpiry) {
	case "", "custom":
		return parseDateTime(customValue)
	case "3d":
		return now.AddDate(0, 0, 3), nil
	case "7d":
		return now.AddDate(0, 0, 7), nil
	case "1m":
		return now.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid quick expiry: %s", quickExpiry)
	}
}
