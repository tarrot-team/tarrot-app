package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/httpapi"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/store"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	var st store.MessageStore
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pgStore, err := store.OpenPostgres(ctx, dsn)
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

		st = pgStore
		log.Info("using postgres storage")
	} else {
		st = store.NewMemoryStore()
		log.Info("DATABASE_URL is empty, using in-memory storage")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
		log.Warn("JWT_SECRET is empty, using development default")
	}

	h := hub.New()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewHandler(h, st, []byte(jwtSecret)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("realtime-service started", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
