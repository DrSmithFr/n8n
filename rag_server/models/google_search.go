package models

type GoogleSearchResult struct {
	Title       string `json:"title"`
	Snippet     string `json:"snippet"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Image       string `json:"image"`
}
