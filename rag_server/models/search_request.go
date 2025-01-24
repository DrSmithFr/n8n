package models

type SearchRequest struct {
	Questions []string `json:"questions"`
	Prompt    string   `json:"prompt"`
	Model     string   `json:"model"`
}

type SearchResponse struct {
	Question string               `json:"question"`
	Links    []GoogleSearchResult `json:"links"`
}
