package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rag_server/models"
)

// GetEmbedding retrieves the embedding vector for the given text
func GetEmbedding(text, model string) ([]float64, error) {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if openAIAPIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is not set")
	}

	url := "https://api.openai.com/v1/embeddings"
	requestBody, err := json.Marshal(models.OpenAIEmbeddingRequest{
		Input: text,
		Model: model,
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
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		return nil, fmt.Errorf("Embedding request failed: %s", body.String())
	}

	var embeddingResponse models.OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResponse); err != nil {
		return nil, fmt.Errorf("Failed to decode embedding response: %v", err)
	}

	if len(embeddingResponse.Data) == 0 {
		return nil, fmt.Errorf("No embedding data returned")
	}

	return embeddingResponse.Data[0].Embedding, nil
}
