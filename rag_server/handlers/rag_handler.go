package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"rag_server/models"
	"rag_server/services"
	"sync"
)

// HandleRAGRequest handles the RAG API requests
func HandleRAGRequest(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		// Apply defaults
		if req.Embedding == "" {
			req.Embedding = "text-embedding-3-small"
		}
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
		responses := make([]models.ResponseItem, len(req.Questions))

		for i, question := range req.Questions {
			wg.Add(1)
			go func(i int, question string) {
				defer wg.Done()
				responses[i] = services.ProcessQuestion(db, question, req.Prompt, req.Embedding, req.Model)
			}(i, question)
		}

		wg.Wait()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responses); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}
