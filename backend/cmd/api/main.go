// Package main は Werewolf Server バックエンドAPIのエントリポイント。
//
// @title           Werewolf Server API
// @version         1.0
// @description     マイクラ人狼 専用バックエンドAPI
// @BasePath        /api
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/mc-werewolf/server/backend/docs"
	"github.com/mc-werewolf/server/backend/internal/api"
	"github.com/mc-werewolf/server/backend/internal/db"
	"github.com/mc-werewolf/server/backend/internal/migrate"
)

func main() {
	devMode := os.Getenv("APP_ENV") == "dev"

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	databaseURL := db.BuildURL(
		getEnv("POSTGRES_HOST", "postgres"),
		getEnv("POSTGRES_PORT", "5432"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	if err := migrate.Up(databaseURL); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	pool, err := db.Connect(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("failed to set up db pool: %v", err)
	}
	defer pool.Close()

	registryURL := getEnv("KAIRO_REGISTRY_URL", "https://kairojs.com")
	addonIDs := strings.Split(getEnv("LAUNCHER_ADDON_IDS", "kairo,kairo-database,game-manager,vanillapack,additional-roles-1"), ",")
	router := api.NewRouter(devMode, pool, api.NewLauncherConfig(registryURL, addonIDs))

	log.Printf("starting server on :%s (devMode=%v)", port, devMode)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
