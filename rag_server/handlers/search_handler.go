package handlers

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"net/http"
	"rag_server/models"
	"rag_server/services"
	"sync"
)

// HandleSearchRequest handles the Web Search API requests
func HandleSearchRequest(rdb *redis.Client, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// Apply defaults
		if req.Model == "" {
			req.Model = "gpt-4o"
		}
		if req.Prompt == "" {
			req.Prompt = `
				You are a RAG model answering questions based on provided documents.
				1. Use only the documents for answers, without personal opinions or extra context. 
				2. End responses with source filenames and URLs: "[ filename ]( url )".
				3. If insufficient information is found, say: "The provided documents do not contain enough information to answer the question."
			`
		}

		var wg sync.WaitGroup
		responses := make([]models.SearchResponse, len(req.Questions))

		for i, question := range req.Questions {
			wg.Add(1)
			go func(i int, question string) {
				defer wg.Done()
				responses[i] = services.ProcessSearch(rdb, ctx, question)
			}(i, question)
		}

		wg.Wait()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responses); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}
