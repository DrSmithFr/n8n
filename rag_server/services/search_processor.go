package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"net/http"
	"rag_server/models"
	"strings"
	"time"
)

// ProcessSearch processes a single question using embeddings and chat
func ProcessSearch(rdb *redis.Client, ctx context.Context, query string) models.SearchResponse {
	cacheKey := fmt.Sprintf("GoogleSearchCached:%s", query)

	// Step 1: Check if the result is already in cache
	cachedData, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		// If cache hit, unmarshal and return
		var cachedResults []models.GoogleSearchResult
		if err := json.Unmarshal([]byte(cachedData), &cachedResults); err == nil {
			fmt.Println("Cache hit: Returning cached results")
			return models.SearchResponse{
				Question: query,
				Links:    cachedResults,
			}
		}
	}

	// Step 2: If not cached, perform the actual Google search
	results := searchGoogle(query)
	if results == nil {
		fmt.Println("No results found from Google API")
		return models.SearchResponse{
			Question: query,
			Links:    []models.GoogleSearchResult{},
		}
	}

	// Step 3: Store the result in cache for future use
	jsonData, err := json.Marshal(results)
	if err == nil {
		err = rdb.Set(ctx, cacheKey, jsonData, 24*time.Hour).Err() // Cache for 24 hours
		if err != nil {
			fmt.Println("Error caching results:", err)
		}
	}

	return models.SearchResponse{
		Question: query,
		Links:    results,
	}
}

func chatGPTCheck(prompt, apiKey, model string) bool {
	payload := map[string]interface{}{
		"model":      model,
		"prompt":     prompt,
		"max_tokens": 5,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/completions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error calling OpenAI API:", err)
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error decoding OpenAI API response:", err)
		return false
	}

	return strings.Contains(result.Choices[0].Text, "Yes")
}
