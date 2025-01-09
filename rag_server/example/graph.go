package main

import (
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"log"
	"os"
	"rag_server/graph"
	"rag_server/graph_builder"
)

type CustomState struct {
	messages []llms.MessageContent
	name     string
}

func (g *CustomState) LastMessage() llms.MessageContent {
	return g.messages[len(g.messages)-1]
}

func main() {
	// Load env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	config := context.Background()
	gb := graph_builder.NewStateGraph[CustomState]()

	model, err := openai.New(
		openai.WithModel("gpt-3.5-turbo"),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
	)
	if err != nil {
		panic(err)
	}

	agent := func(g CustomState, config context.Context) (CustomState, error) {
		r, err := model.GenerateContent(config, g.messages, llms.WithTemperature(0.7))
		if err != nil {
			return CustomState{
				name: g.name,
				messages: []llms.MessageContent{
					llms.TextParts(llms.ChatMessageTypeAI, fmt.Sprintf("Failed to generate answer: %v", err)),
				},
			}, err
		}

		return CustomState{
			name: g.name,
			messages: []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeAI, r.Choices[0].Content),
			},
		}, nil
	}

	gb.AddNode("agent", agent)

	gb.SetEntryPoint("agent")
	gb.AddEdge("agent", graph.END)

	g, err := gb.Compile()
	if err != nil {
		log.Fatal(err)
	}

	input := CustomState{
		name: "",
		messages: []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "What is 1 + 1?"),
		},
	}

	states, err := g.Stream(input, config)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(states[len(states)-1].State.messages)
}
