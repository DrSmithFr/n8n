package models

type RagRequest struct {
	Questions []string `json:"questions"`
	Prompt    string   `json:"prompt"`
	Embedding string   `json:"embedding"`
	Model     string   `json:"model"`
}

type RagResponseItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}
