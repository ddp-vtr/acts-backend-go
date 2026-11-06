package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"acts-backend-go/internal/database"
	"acts-backend-go/internal/model"
	"acts-backend-go/internal/rag"
)

func main() {
	repo := database.NewRepository()
	endpoint := os.Getenv("ROUTER_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://rlau4p8.abc-tunnel.us/v1/chat/completions"
	}
	apiKey := os.Getenv("ROUTER_API_KEY")
	if apiKey == "" {
		apiKey = "sk-5b2c9a975f7f309f-qcuwzl-e7e1385d"
	}
	modelName := os.Getenv("ROUTER_MODEL")
	if modelName == "" {
		modelName = "ag/gemini-3.8-flash-high"
	}

	engine := rag.NewEngine(repo, endpoint, apiKey, modelName)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "service": "acts-backend-go"})
	})

	mux.HandleFunc("POST /api/v1/chat", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req model.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, `{"error":"message_required"}`, http.StatusBadRequest)
			return
		}

		resp, err := engine.ProcessQuery(req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	log.Printf("acts-backend-go running on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
