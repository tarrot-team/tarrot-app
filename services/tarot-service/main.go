package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/httpapi"
	"github.com/tarrot-team/tarrot-app/services/tarot-service/internal/interpreter"
)

func main() {
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
		log.Printf("tarot-service LLM enabled (%s, model=%s)", llmBaseURL, llmModel)
	} else {
		log.Println("tarot-service running with fallback interpreter (set GROQ_API_KEY or LLM_API_KEY to enable AI)")
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewHandler(interp),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("tarot-service listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
