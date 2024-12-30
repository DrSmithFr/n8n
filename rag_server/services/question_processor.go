package services

import (
	"database/sql"
	"fmt"
	"rag_server/models"
)

// ProcessQuestion processes a single question using embeddings and chat
func ProcessQuestion(db *sql.DB, question, prompt, embeddingModel, chatModel string) models.ResponseItem {
	// Step 1: Generate embedding for the question
	embedding, err := GetEmbedding(question, embeddingModel)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate embedding for question '%s': %v", question, err)
		return models.ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
	}

	// Step 2: Query the database for related documents
	contextItems, err := SearchItems(db, embedding)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to fetch context items for question '%s': %v", question, err)
		return models.ResponseItem{
			Question: question,
			Answer:   logMessage,
		}
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
