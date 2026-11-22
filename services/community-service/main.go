package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tarrot-team/tarrot-app/services/community-service/internal/httpapi"
	"github.com/tarrot-team/tarrot-app/services/community-service/internal/profile"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	var store profile.Store
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pgStore, err := profile.OpenPostgres(ctx, dsn)
		cancel()
		if err != nil {
			log.Error("failed to connect to postgres", "err", err)
			os.Exit(1)
		}
		defer func() { _ = pgStore.Close() }()

		schemaCtx, schemaCancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = pgStore.EnsureSchema(schemaCtx)
		schemaCancel()
		if err != nil {
			log.Error("failed to ensure database schema", "err", err)
			os.Exit(1)
		}

		store = pgStore
		log.Info("using postgres storage")
	} else {
		store = profile.NewMemoryStore()
		log.Info("DATABASE_URL is empty, using in-memory storage")
	}

	svc := profile.NewService(store)
	h := httpapi.NewHandler(svc, log)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("community-service started", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
