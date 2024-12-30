package models

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
