package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"rag_server/models"
	"strconv"
)

// SearchItems récupère les documents similaires à partir de la base de données
func SearchItems(db *sql.DB, embedding []float64) ([]models.ContextItem, error) {
	const query = `
		SELECT text, metadata
		FROM n8n_vectors
		ORDER BY embedding <=> $1
		LIMIT 10;
	`

	rows, err := db.Query(query, ToVectorString(embedding))
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	var contextItems []models.ContextItem
	for rows.Next() {
		var text string
		var metadataRaw json.RawMessage
		if err := rows.Scan(&text, &metadataRaw); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		contextItems = append(contextItems, models.ContextItem{
			Text:     text,
			Metadata: metadataRaw,
		})
	}

	return contextItems, nil
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
