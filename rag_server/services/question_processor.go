package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"rag_server/models"
	"strconv"
)

// ProcessQuestion processes a single question using embeddings and chat
func ProcessQuestion(db *sql.DB, question, prompt, embeddingModel, chatModel string) models.ResponseItem {
	const query = `
		SELECT text, metadata
		FROM n8n_vectors
		ORDER BY embedding <=> $1
		LIMIT 10;
	`

	// Step 1: Generate embedding for the question
	embedding, err := GetEmbedding(question, embeddingModel)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate embedding for question '%s': %v", question, err)
		return models.ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}

	vectorString := ToVectorString(embedding)

	// Step 2: Query the database for related documents
	rows, err := db.Query(query, vectorString)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to query database for question '%s': %v", question, err)
		return models.ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}
	defer rows.Close()

	var contextItems []struct {
		Text     string
		Metadata json.RawMessage
	}
	for rows.Next() {
		var text string
		var metadataRaw json.RawMessage
		if err := rows.Scan(&text, &metadataRaw); err != nil {
			log.Printf("Failed to scan row for question '%s': %v", question, err)
			continue
		}

		contextItems = append(contextItems, struct {
			Text     string
			Metadata json.RawMessage
		}{
			Text:     text,
			Metadata: metadataRaw,
		})
	}

	if len(contextItems) == 0 {
		return models.ResponseItem{
			Question: question,
			Answer:   "The database does not contain enough information to answer the question.",
		}
	}

	// Step 3: Prepare context for the question
	var context []string
	for _, item := range contextItems {
		context = append(context, fmt.Sprintf("Text: %s\nMetadata: %s", item.Text, item.Metadata))
	}

	// Step 4: Generate an answer using the context and OpenAI API
	answer, err := GenerateAnswer(question, context, prompt, chatModel)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate answer for question '%s': %v", question, err)
		return models.ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}

	// Step 5: Return the final response
	return models.ResponseItem{
		Question: question,
		Answer:   answer,
	}
}

// ToVectorString converts a slice of floats into a PostgreSQL-compatible vector string
func ToVectorString(data []float64) string {
	buf := make([]byte, 0, 2+16*len(data))
	buf = append(buf, '[')

	for i, value := range data {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = strconv.AppendFloat(buf, value, 'f', -1, 64)
	}

	buf = append(buf, ']')
	return string(buf)
}
