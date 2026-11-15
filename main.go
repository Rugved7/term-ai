package main

import (
	"bufio"
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

	client := ollama.NewClient()
	ctx := context.Background()

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

		// If a prompt is supplied, execute one-shot chat.
		prompt := strings.Join(chatCmd.Args(), " ")

		if prompt != "" {
			request := llm.ChatRequest{
				Model: *model,
				Messages: []llm.Message{
					{Role: "user", Content: prompt},
				},
			}

			chatCtx, cancel := context.WithTimeout(
				ctx, 5*time.Minute,
			)
			defer cancel()

			response, err := client.Chat(chatCtx, request)
			if err != nil {
				exitWithError(err)
			}

			fmt.Println("\nTermAI:")
			fmt.Println(response.Message.Content)
			return
		}

		// Otherwise start interactive chat.
		startChat(client, *model)

	default:
		printUsage()
	}
}

func startChat(client *ollama.Client, model string) {
	scanner := bufio.NewScanner(os.Stdin)

	// Allow inputs up to 1 MB.
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	// Conversation history.
	history := []llm.Message{}

	fmt.Println("\nWelcome to TermAI!")
	fmt.Println("Provider:", "Ollama")
	fmt.Println("Model:", model)
	fmt.Println("Type /exit to quit, /clear to reset history.")
	fmt.Println("--------------------------------------------")

	for {
		fmt.Print("\nYou > ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		// Handle commands.
		switch input {
		case "/exit":
			fmt.Println("Goodbye!")
			return

		case "/clear":
			history = []llm.Message{}
			fmt.Println("Conversation history cleared.")
			continue
		}

		// Add user message to history.
		history = append(history, llm.Message{
			Role:    "user",
			Content: input,
		})

		// Create request.
		request := llm.ChatRequest{
			Model:    model,
			Messages: history,
		}

		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Minute,
		)

		// response, err := client.Chat(ctx, request)
		// cancel()

		// if err != nil {
		// 	fmt.Println("Error:", err)

		// 	// Remove failed user message.
		// 	history = history[:len(history)-1]
		// 	continue
		// }

		// // Add assistant response to history.
		// history = append(history, response.Message)

		fmt.Print("\nTermAI > ")

		response, err := client.ChatStream(
			ctx,
			request,
			func(chunk string) {
				fmt.Print(chunk)
			},
		)

		cancel()

		fmt.Println()

		if err != nil {
			fmt.Println("Error:", err)

			history = history[:len(history)-1]
			continue
		}

		history = append(history, response.Message)

		// fmt.Println("\nTermAI >")
		// fmt.Println(response.Message.Content)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Input error:", err)
	}

	fmt.Println("\nGoodbye!")
}

func printUsage() {
	fmt.Println(`
TermAI - Your local AI terminal client

Usage:
  termai models
  termai chat
  termai chat "your prompt"
  termai chat -m llama3.2:3b
`)
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
