package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Rugved7/termai/internal/llm"
	"github.com/Rugved7/termai/internal/provider/ollama"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	ctx := context.Background()
	client := ollama.NewClient()

	switch os.Args[1] {

	case "models":
		models, err := client.Models(ctx)
		if err != nil {
			exitWithError(err)
		}

		fmt.Println("Available models:")
		for _, model := range models {
			fmt.Println(" -", model)
		}

	case "chat":
		chatCmd := flag.NewFlagSet("chat", flag.ExitOnError)

		model := chatCmd.String(
			"m",
			"llama3.2:3b",
			"Model to use",
		)

		if err := chatCmd.Parse(os.Args[2:]); err != nil {
			exitWithError(err)
		}

		prompt := strings.Join(chatCmd.Args(), " ")

		if strings.TrimSpace(prompt) == "" {
			fmt.Println("Please provide a prompt.")
			os.Exit(1)
		}

		requestCtx, cancel := context.WithTimeout(
			ctx,
			5*time.Minute,
		)
		defer cancel()

		request := llm.ChatRequest{
			Model: *model,
			Messages: []llm.Message{
				{
					Role:    "user",
					Content: prompt,
				},
			},
		}

		response, err := client.Chat(requestCtx, request)
		if err != nil {
			exitWithError(err)
		}

		fmt.Println("\nTermAI:")
		fmt.Println(response.Message.Content)

	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`
TermAI - Your local AI terminal client

Usage:
  termai models
  termai chat "your prompt"
  termai chat -m llama3.2:3b "your prompt"
`)
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
