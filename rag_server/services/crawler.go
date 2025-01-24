package services

import (
	"fmt"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"io"
	"log"
	"net/http"
)

func RetrieveUrlContents(url string) (string, error) {
	// Fetch the URL
	htmlContent, err := crawl(url)
	if err != nil {
		return "", fmt.Errorf("error fetching URL: %v", err)
	}

	markdown, err := htmltomarkdown.ConvertString(htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	return markdown, nil
}

func crawl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Error fetching HTML content: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body: %v", err)
	}

	return string(body), nil
}
