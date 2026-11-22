package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/httpapi"
	"github.com/tarrot-team/tarrot-app/services/realtime-service/internal/hub"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	h := hub.New()
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewHandler(h),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("realtime-service started", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
