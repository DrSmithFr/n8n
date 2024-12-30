package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	_ "github.com/lib/pq"
)

type Request struct {
	Questions []string `json:"questions"`
}

type ResponseItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

var db *sql.DB

func main() {
	var err error

	// Initialize database connection
	db, err = sql.Open("postgres", fmt.Sprintf(
		"host=postgres user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"),
	))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/api/rag", handleRAGRequest)
	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleRAGRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	var wg sync.WaitGroup
	responses := make([]ResponseItem, len(req.Questions))

	for i, question := range req.Questions {
		wg.Add(1)
		go func(i int, question string) {
			defer wg.Done()
			responses[i] = processQuestion(question)
		}(i, question)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func processQuestion(question string) ResponseItem {
	const query = `
		SELECT text
		FROM n8n_vectors
		WHERE embedding <=> $1
		ORDER BY embedding <=> $1
		LIMIT 1;
	`

	embedding, err := getEmbedding(question)
	if err != nil {
		log.Printf("Failed to generate embedding for question '%s': %v", question, err)
		return ResponseItem{
			Question: question,
			Answer:   "The provided documents do not contain enough information to answer the question.",
		}
	}

	var answer string
	row := db.QueryRow(query, embedding)
	if err := row.Scan(&answer); err != nil {
		if err == sql.ErrNoRows {
			return ResponseItem{
				Question: question,
				Answer:   "The provided documents do not contain enough information to answer the question.",
			}
		}
		log.Printf("Failed to query database for question '%s': %v", question, err)
		return ResponseItem{
			Question: question,
			Answer:   "The provided documents do not contain enough information to answer the question.",
		}
	}

	return ResponseItem{
		Question: question,
		Answer:   answer,
	}
}

func getEmbedding(text string) ([]float64, error) {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if openAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not set")
	}

	url := "https://api.openai.com/v1/embeddings"
	requestBody, err := json.Marshal(OpenAIEmbeddingRequest{
		Input: text,
		Model: "text-embedding-ada-002",
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal embedding request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("Failed to create embedding request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openAIAPIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to execute embedding request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Embedding request failed: %s", string(body))
	}

	var embeddingResponse OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResponse); err != nil {
		return nil, fmt.Errorf("Failed to decode embedding response: %v", err)
	}

	if len(embeddingResponse.Data) == 0 {
		return nil, fmt.Errorf("No embedding data returned")
	}

	return embeddingResponse.Data[0].Embedding, nil
}
