package httpapp

import (
	"context"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"4vpx/internal/backup"
	"4vpx/internal/config"
	"4vpx/internal/domain"
	"4vpx/internal/http/handlers"
	"4vpx/internal/http/middleware"
	"4vpx/internal/service"
	"4vpx/internal/storage/sqlite"
	"4vpx/internal/xray"
)

func NewRouter(ctx context.Context, db *sql.DB, cfg config.Config) (http.Handler, error) {
	store := sqlite.NewStore(db)
	defaultSystem := domain.SystemConfig{
		ID:                1,
		ServerAddress:     cfg.ServerAddress,
		ServerPort:        cfg.ServerPort,
		RealityDest:       cfg.RealityDest,
		RealityServerName: cfg.RealityServerName,
		ClientFingerprint: cfg.ClientFingerprint,
		RealityPrivateKey: cfg.RealityPrivateKey,
		RealityPublicKey:  cfg.RealityPublicKey,
		RealityShortID:    cfg.RealityShortID,
		XrayLogLevel:      cfg.XrayLogLevel,
		XrayConfigPath:    cfg.XrayConfigPath,
		XrayBackupPath:    cfg.XrayBackupPath,
		XrayBin:           cfg.XrayBin,
		XrayReloadCmd:     cfg.XrayReloadCmd,
		UpdatedAt:         time.Now().UTC(),
	}

	adminService := service.NewAdminService(store)
	if _, err := adminService.EnsureInitialized(ctx, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		return nil, err
	}
	systemService := service.NewSystemService(store, defaultSystem)
	if _, err := systemService.Ensure(ctx); err != nil {
		return nil, err
	}

	userService := service.NewUserService(store)
	deviceService := service.NewDeviceService(store)
	renewalService := service.NewRenewalService(store)
	renderer, err := xray.NewRenderer()
	if err != nil {
		return nil, err
	}
	runtime := xray.NewRuntime(renderer)
	publishService := service.NewPublishService(store, systemService, runtime)
	portalService := service.NewUserPortalService(userService, deviceService, renewalService, systemService, renderer)

	tpl, err := template.ParseGlob("web/templates/*.html")
	if err != nil {
		return nil, err
	}

	app := &handlers.App{
		Templates:      tpl,
		SessionManager: middleware.NewSessionManager(store.Sessions, cfg.SessionCookieName, cfg.SessionSecure),
		CSRFManager:    middleware.NewCSRFManager(cfg.SessionSecure),
		AdminService:   adminService,
		UserService:    userService,
		DeviceService:  deviceService,
		RenewalService: renewalService,
		SystemService:  systemService,
		PublishService: publishService,
		PortalService:  portalService,
		Exporter:       backup.NewExporter(db),
		Importer:       backup.NewImporter(db),
		BaseURL:        cfg.AppBaseURL,
	}
	startAutoPublish(ctx, publishService, cfg.AutoPublishInterval)

	mux := http.NewServeMux()
	loginHandler := app.CSRFManager.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			app.LoginPage(w, r)
			return
		}
		if r.Method == http.MethodPost {
			app.LoginSubmit(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	})
	mux.Handle("/login", loginHandler)
	mux.Handle("/logout", app.CSRFManager.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			app.Logout(w, r)
			return
		}
		http.NotFound(w, r)
	})))
	mux.HandleFunc("/u/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/u/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		token := parts[0]
		if len(parts) == 1 && r.Method == http.MethodGet {
			app.UserPortal(w, r, token)
			return
		}
		if len(parts) == 4 && parts[1] == "devices" && r.Method == http.MethodGet {
			slotIndex, err := handlers.ParseSlotIndex(parts[2])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			if parts[3] == "mihomo" {
				app.UserDeviceMihomo(w, r, token, slotIndex)
				return
			}
			if parts[3] == "vless" {
				app.UserDeviceVLESS(w, r, token, slotIndex)
				return
			}
		}
		http.NotFound(w, r)
	})

	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/admin/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			app.ListUsers(w, r)
			return
		}
		if r.Method == http.MethodPost {
			app.CreateUser(w, r)
			return
		}
		http.NotFound(w, r)
	})
	adminMux.HandleFunc("/admin/system", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			app.SystemPage(w, r)
			return
		}
		if r.Method == http.MethodPost {
			app.UpdateSystem(w, r)
			return
		}
		http.NotFound(w, r)
	})
	adminMux.HandleFunc("/admin/change-password", app.ChangePassword)
	adminMux.HandleFunc("/admin/export", app.ExportBackup)
	adminMux.HandleFunc("/admin/import", app.ImportBackup)
	adminMux.HandleFunc("/admin/users/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/admin/users/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		userID, err := handlers.ParseUserID(parts[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if len(parts) == 1 && r.Method == http.MethodGet {
			app.ShowUserDetail(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "update" && r.Method == http.MethodPost {
			app.UpdateUser(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "toggle" && r.Method == http.MethodPost {
			app.ToggleUser(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "delete" && r.Method == http.MethodPost {
			app.DeleteUser(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "renew-7" && r.Method == http.MethodPost {
			app.Renew7Days(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "renew-month" && r.Method == http.MethodPost {
			app.Renew1Month(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "renew-custom" && r.Method == http.MethodPost {
			app.RenewCustom(w, r, userID)
			return
		}
		if len(parts) == 2 && parts[1] == "devices" && r.Method == http.MethodPost {
			app.AdjustDevices(w, r, userID)
			return
		}
		if len(parts) == 4 && parts[1] == "devices" && parts[3] == "reset" && r.Method == http.MethodPost {
			slotIndex, err := handlers.ParseSlotIndex(parts[2])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			app.ResetDeviceUUID(w, r, userID, slotIndex)
			return
		}
		if len(parts) == 4 && parts[1] == "devices" && parts[3] == "toggle" && r.Method == http.MethodPost {
			slotIndex, err := handlers.ParseSlotIndex(parts[2])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			app.ToggleDevice(w, r, userID, slotIndex)
			return
		}
		http.NotFound(w, r)
	})

	mux.Handle("/admin/", app.SessionManager.Require(app.CSRFManager.Protect(adminMux)))
	return mux, nil
}

func startAutoPublish(ctx context.Context, publishService *service.PublishService, interval time.Duration) {
	if interval <= 0 {
		log.Printf("auto publish disabled")
		return
	}

	log.Printf("auto publish enabled: interval=%s", interval)
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := publishService.Publish(context.Background())
				if err != nil {
					log.Printf("auto publish failed: %v", err)
					continue
				}
				log.Printf("auto publish complete: active_clients=%d reloaded=%t config=%s", result.ActiveClients, result.Reloaded, result.ConfigPath)
			}
		}
	}()
}
