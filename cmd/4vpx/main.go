package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"4vpx/internal/config"
	httpapp "4vpx/internal/http"
	"4vpx/internal/storage/sqlite"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	_ = os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755)
	_ = os.MkdirAll(filepath.Dir(cfg.XrayConfigPath), 0o755)
	_ = os.MkdirAll(filepath.Dir(cfg.XrayBackupPath), 0o755)

	db, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router, err := httpapp.NewRouter(context.Background(), db, cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("4vpx listening on %s", cfg.AppAddr)
	if err := http.ListenAndServe(cfg.AppAddr, router); err != nil {
		log.Fatal(err)
	}
}
