package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rag_server/models"
)

// GenerateAnswer queries the OpenAI chat model to generate an answer for the question
func GenerateAnswer(question string, context []string, prompt, model string) (string, error) {
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

	requestBody, err := json.Marshal(models.OpenAIChatRequest{
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
		body := new(bytes.Buffer)
		body.ReadFrom(resp.Body)
		return "", fmt.Errorf("Chat request failed: %s", body.String())
	}

	var chatResponse models.OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		return "", fmt.Errorf("Failed to decode chat response: %v", err)
	}

	if len(chatResponse.Choices) == 0 || chatResponse.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("No chat response returned")
	}

	return chatResponse.Choices[0].Message.Content, nil
}
