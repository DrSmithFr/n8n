package models

import "encoding/json"

type ContextItem struct {
	Text     string          `json:"text"`
	Metadata json.RawMessage `json:"metadata"`
}
