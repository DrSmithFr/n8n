package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"rag_server/db"
	"rag_server/handlers"
)

func main() {
	// Load env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database connection
	dbConn, err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Set up HTTP handlers
	http.HandleFunc("/api/rag", handlers.HandleRAGRequest(dbConn))

	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
