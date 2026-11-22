package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/httpapi"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/interpreter"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/reading"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := envOr("PORT", "8081")
	llmBaseURL := envOr("LLM_BASE_URL", "https://api.groq.com/openai/v1")
	llmModel := envOr("LLM_MODEL", "llama-3.3-70b-versatile")
	llmAPIKey := os.Getenv("LLM_API_KEY")
	if llmAPIKey == "" {
		llmAPIKey = os.Getenv("GROQ_API_KEY")
	}

	var interp interpreter.Interpreter = interpreter.FallbackInterpreter{}
	if llmAPIKey != "" || os.Getenv("LLM_BASE_URL") != "" {
		interp = interpreter.WithFallback{
			Primary:  interpreter.NewOpenAICompatClient(llmBaseURL, llmAPIKey, llmModel),
			Fallback: interpreter.FallbackInterpreter{},
		}
		log.Info("LLM enabled", "base_url", llmBaseURL, "model", llmModel)
	} else {
		log.Info("running with fallback interpreter")
	}

	var store reading.Store
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pgStore, err := reading.OpenPostgres(ctx, dsn)
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
		store = reading.NewMemoryStore()
		log.Info("DATABASE_URL is empty, using in-memory storage")
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewHandler(interp, store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Info("tarot-service started", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
