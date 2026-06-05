package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"4vpx/internal/service"
)

func (a *App) SystemPage(w http.ResponseWriter, r *http.Request) {
	cfg, err := a.SystemService.Get(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.render(w, r, TemplateData{
		Title:           "System",
		ContentTemplate: "admin_system_content",
		CurrentPath:     "/admin/system",
		System:          cfg,
		Flash:           a.flash(r),
		Error:           a.errText(r),
	})
}

func (a *App) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", "表单解析失败")
		return
	}
	serverPort, err := parseInt(r.FormValue("server_port"))
	if err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", "端口格式不正确")
		return
	}
	_, err = a.SystemService.Update(r.Context(), service.UpdateSystemInput{
		ServerAddress:     r.FormValue("server_address"),
		ServerPort:        serverPort,
		RealityDest:       r.FormValue("reality_dest"),
		RealityServerName: r.FormValue("reality_server_name"),
		ClientFingerprint: r.FormValue("client_fingerprint"),
		RealityPrivateKey: r.FormValue("reality_private_key"),
		RealityPublicKey:  r.FormValue("reality_public_key"),
		RealityShortID:    r.FormValue("reality_short_id"),
		XrayLogLevel:      r.FormValue("xray_loglevel"),
		XrayConfigPath:    r.FormValue("xray_config_path"),
		XrayBackupPath:    r.FormValue("xray_backup_path"),
		XrayBin:           r.FormValue("xray_bin"),
		XrayReloadCmd:     r.FormValue("xray_reload_cmd"),
	})
	if err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "系统配置已保存"); err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", err.Error())
		return
	}
	redirectWithMessage(w, r, "/admin/system", "flash", "系统配置已保存并发布")
}

func (a *App) ExportBackup(w http.ResponseWriter, r *http.Request) {
	data, err := a.Exporter.ExportJSON(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"4vpx-backup-"+time.Now().UTC().Format("20060102-150405")+".json\"")
	_, _ = w.Write(data)
}

func (a *App) ImportBackup(w http.ResponseWriter, r *http.Request) {
	payload, err := readBackupPayload(r)
	if err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", err.Error())
		return
	}
	if err := a.Importer.ImportJSON(r.Context(), payload); err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", err.Error())
		return
	}
	if err := a.publishMutation(r, "备份已导入"); err != nil {
		redirectWithMessage(w, r, "/admin/system", "error", err.Error())
		return
	}
	redirectWithMessage(w, r, "/admin/system", "flash", "备份已导入并发布")
}

func readBackupPayload(r *http.Request) ([]byte, error) {
	if err := r.ParseMultipartForm(8 << 20); err == nil {
		file, _, fileErr := r.FormFile("backup_file")
		if fileErr == nil {
			defer file.Close()
			payload, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}
			if len(bytes.TrimSpace(payload)) == 0 {
				return nil, fmt.Errorf("请上传备份 JSON 文件或粘贴 JSON 内容")
			}
			return payload, nil
		}
	}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("表单解析失败")
	}
	payload := bytes.TrimSpace([]byte(r.FormValue("backup_json")))
	if len(payload) == 0 {
		return nil, fmt.Errorf("请上传备份 JSON 文件或粘贴 JSON 内容")
	}
	return payload, nil
}
