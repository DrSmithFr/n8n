package services

import (
	"encoding/json"
	"fmt"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"net/http"
	"os"
	"rag_server/cache"
	"rag_server/models"
	"strings"
	"time"
	"unicode"
)

// queryToParameter replaces spaces with '+', removes accents, and removes special characters
func queryToParameter(query string) string {
	// Replace spaces with "+"
	query = strings.ReplaceAll(query, " ", "+")

	// Remove accents and normalize the string
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isNonSpacingMark), norm.NFC)
	normalized, _, _ := transform.String(t, query)

	// Remove special characters
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' {
			return r
		}
		return -1
	}, normalized)

	return cleaned
}

// Helper function to identify non-spacing marks (used to remove accents)
func isNonSpacingMark(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

// searchGoogle performs the actual Google Custom Search API call
func searchGoogle(query string) []models.GoogleSearchResult {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	cseId := os.Getenv("CSE_ID")
	url := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?q=%s&key=%s&cx=%s", queryToParameter(query), apiKey, cseId)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error during Google Custom Search:", err)
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Link    string `json:"link"`
			PageMap struct {
				Metas []map[string]string `json:"metatags"`
			} `json:"pagemap"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return nil
	}

	var links []models.GoogleSearchResult
	for _, item := range result.Items {
		link := models.GoogleSearchResult{
			Title:   item.Title,
			Snippet: item.Snippet,
			Url:     item.Link,
		}

		if len(item.PageMap.Metas) > 0 {
			metas := item.PageMap.Metas[0]

			if title, ok := metas["og:title"]; ok {
				link.Title = title
			}

			if desc, ok := metas["og:description"]; ok {
				link.Description = desc
			}

			if img, ok := metas["og:image"]; ok {
				link.Image = img
			}
		}

		links = append(links, link)
	}

	return links
}

// SearchGoogleCached searches Google and caches the result in Redis
func SearchGoogleCached(query string) []models.GoogleSearchResult {
	rdb, ctx := cache.InitCache()
	cacheKey := fmt.Sprintf("GoogleSearchCached:%s", query)

	// Step 1: Check if the result is already in cache
	cachedData, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		// If cache hit, unmarshal and return
		var cachedResults []models.GoogleSearchResult
		if err := json.Unmarshal([]byte(cachedData), &cachedResults); err == nil {
			fmt.Println("Cache hit: Returning cached results")
			return cachedResults
		}
	}

	// Step 2: If not cached, perform the actual Google search
	results := searchGoogle(query)
	if results == nil {
		fmt.Println("No results found from Google API")
		return nil
	}

	// Step 3: Store the result in cache for future use
	jsonData, err := json.Marshal(results)
	if err == nil {
		err = rdb.Set(ctx, cacheKey, jsonData, 24*time.Hour).Err() // Cache for 24 hours
		if err != nil {
			fmt.Println("Error caching results:", err)
		}
	}

	return results
}
