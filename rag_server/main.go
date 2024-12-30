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
	"strconv"
	"sync"

	_ "github.com/lib/pq"
)

type Request struct {
	Questions []string `json:"questions"`
	Prompt    string   `json:"prompt"`
	Embedding string   `json:"embedding"`
	Model     string   `json:"model"`
}

type ResponseItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`

	EncodingFormat string `json:"encoding_format"`
}

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

type OpenAIChatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var db *sql.DB

func main() {
	var err error

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "postgres"
	}

	// Initialize database connection
	db, err = sql.Open("postgres", fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s sslmode=disable",
		host,
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
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

	if req.Embedding == "" {
		req.Embedding = "text-embedding-3-large"
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
	responses := make([]ResponseItem, len(req.Questions))

	for i, question := range req.Questions {
		wg.Add(1)
		go func(i int, question string) {
			defer wg.Done()
			responses[i] = processQuestion(question, req.Prompt, req.Embedding, req.Model)
		}(i, question)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func processQuestion(question, prompt, embeddingModel, chatModel string) ResponseItem {
	const query = `
		SELECT text
		FROM n8n_vectors
		ORDER BY embedding <=> $1
		LIMIT 10;
	`

	embedding, err := getEmbedding(question, embeddingModel)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate embedding for question '%s': %v", question, err)
		log.Println(logMessage)
		return ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}

	rows, err := db.Query(query, ToVectorString(embedding))
	if err != nil {
		logMessage := fmt.Sprintf("Failed to query database for question: %v", err)
		log.Println(logMessage)
		return ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}
	defer rows.Close()

	var contextTexts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			logMessage := fmt.Sprintf("Failed to scan row for question '%s': %v", question, err)
			log.Println(logMessage)
			continue
		}
		contextTexts = append(contextTexts, text)
	}

	if len(contextTexts) == 0 {
		return ResponseItem{
			Question: question,
			Answer:   "The database do not contain enough information to answer the question.",
		}
	}

	answer, err := generateAnswer(question, contextTexts, prompt, chatModel)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate answer for question '%s': %v", question, err)
		log.Println(logMessage)
		return ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}

	return ResponseItem{
		Question: question,
		Answer:   answer,
	}
}

func ToVectorString(data []float64) string {
	buf := make([]byte, 0, 2+16*len(data))
	buf = append(buf, '[')

	for i := 0; i < len(data); i++ {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendFloat(buf, float64(data[i]), 'f', -1, 32)
	}

	buf = append(buf, ']')
	return string(buf)
}

func getEmbedding(text, model string) ([]float64, error) {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if openAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not set")
	}

	url := "https://api.openai.com/v1/embeddings"
	requestBody, err := json.Marshal(OpenAIEmbeddingRequest{
		Input:          text,
		Model:          model,
		EncodingFormat: "float",
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

	return embeddingResponse.Data[0].Embedding[:1536], nil
}

func generateAnswer(question string, context []string, prompt, model string) (string, error) {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if openAIAPIKey == "" {
		return "", fmt.Errorf("OpenAI API key is not set")
	}

	url := "https://api.openai.com/v1/chat/completions"
	messages := []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{
		{Role: "system", Content: prompt},
		{Role: "user", Content: fmt.Sprintf("Question: %s\nContext: %s", question, context)},
	}
	requestBody, err := json.Marshal(OpenAIChatRequest{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("Failed to marshal chat request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("Failed to create chat request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openAIAPIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to execute chat request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("Chat request failed: %s", string(body))
	}

	var chatResponse OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return "", fmt.Errorf("Failed to decode chat response: %v", err)
	}

	if len(chatResponse.Choices) == 0 || chatResponse.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("No chat response returned")
	}

	return chatResponse.Choices[0].Message.Content, nil
}
