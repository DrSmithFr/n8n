package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"rag_server/cache"
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

	// Initialize Redis cache
	rdb, ctx := cache.InitCache()
	defer rdb.Close()

	// Set up HTTP handlers
	http.HandleFunc("/api/rag", handlers.HandleRAGRequest(dbConn))
	http.HandleFunc("/api/search", handlers.HandleSearchRequest(rdb, ctx))

	log.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
