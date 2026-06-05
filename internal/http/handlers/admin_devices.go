package handlers

import (
	"fmt"
	"net/http"
)

func (a *App) AdjustDevices(w http.ResponseWriter, r *http.Request, userID int64) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "表单解析失败")
		return
	}
	count, err := parseInt(r.FormValue("device_slots"))
	if err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "设备位数量格式不正确")
		return
	}
	if _, err := a.DeviceService.AdjustSlotCount(r.Context(), userID, count); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "设备位数量已更新"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "设备位数量已更新")
}

func (a *App) ResetDeviceUUID(w http.ResponseWriter, r *http.Request, userID int64, slotIndex int) {
	if _, err := a.DeviceService.ResetSlotUUID(r.Context(), userID, slotIndex); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "设备位 UUID 已重置"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "设备位 UUID 已重置")
}

func (a *App) ToggleDevice(w http.ResponseWriter, r *http.Request, userID int64, slotIndex int) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", "表单解析失败")
		return
	}
	enabled := r.FormValue("enabled") == "1"
	if _, err := a.DeviceService.SetSlotEnabled(r.Context(), userID, slotIndex, enabled); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "设备位状态已更新"); err != nil {
		redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "error", err.Error())
		return
	}
	redirectWithMessage(w, r, fmt.Sprintf("/admin/users/%d", userID), "flash", "设备位状态已更新")
}
