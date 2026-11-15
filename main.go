package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Rugved7/termai/internal/conversation"
	"github.com/Rugved7/termai/internal/llm"
	"github.com/Rugved7/termai/internal/storage"
)

// func main() {
// 	if len(os.Args) < 2 {
// 		printUsage()
// 		return
// 	}

// 	client := ollama.NewClient()
// 	ctx := context.Background()

// 	switch os.Args[1] {
// 	case "models":
// 		models, err := client.Models(ctx)
// 		if err != nil {
// 			exitWithError(err)
// 		}

// 		fmt.Println("Available models:")
// 		for _, model := range models {
// 			fmt.Println(" -", model)
// 		}

// 	case "chat":
// 		chatCmd := flag.NewFlagSet("chat", flag.ExitOnError)

// 		model := chatCmd.String("m", "llama3.2:3b", "Model to use")

// 		if err := chatCmd.Parse(os.Args[2:]); err != nil {
// 			exitWithError(err)
// 		}

// 		prompt := strings.Join(chatCmd.Args(), " ")

// 		// One-shot chat.
// 		if prompt != "" {
// 			request := llm.ChatRequest{
// 				Model: *model,
// 				Messages: []llm.Message{
// 					{Role: "user", Content: prompt},
// 				},
// 			}

// 			chatCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
// 			defer cancel()

// 			response, err := client.Chat(chatCtx, request)
// 			if err != nil {
// 				exitWithError(err)
// 			}

// 			fmt.Println("\nTermAI:")
// 			fmt.Println(response.Message.Content)
// 			return
// 		}

// 		// Initialize persistent storage.
// 		homeDir, err := os.UserHomeDir()
// 		if err != nil {
// 			exitWithError(err)
// 		}

// 		dataDir := filepath.Join(homeDir, ".termai")

// 		store, err := sqlite.New(dataDir)
// 		if err != nil {
// 			exitWithError(err)
// 		}

// 		defer func() {
// 			if err := store.Close(); err != nil {
// 				fmt.Fprintln(os.Stderr, "Database close error:", err)
// 			}
// 		}()

// 		startChat(client, *model, store)

// 	default:
// 		printUsage()
// 	}
// }

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func startChat(provider llm.Provider, model string, store storage.Store) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	ctx := context.Background()

	session, err := createChatSession(ctx, store, model)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to create chat session:", err)
		return
	}

	history := []llm.Message{}

	fmt.Print(`
 _____                   ___
|_   _|__ _ __ _ __ ___ |_ _|
  | |/ _ \ '__| '_ ' _ \ | |
  | |  __/ |  | | | | | || |
  |_|\___|_|  |_| |_| |_|___|

Welcome to TermAI
`)

	fmt.Println("Provider :", "Ollama")
	fmt.Println("Model    :", model)
	fmt.Println("Session  :", session.ID)

	printChatCommands()

	for {
		fmt.Print("\nYou > ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch {
		case input == "/exit":
			fmt.Println("Goodbye!")
			return

		case input == "/clear":
			newSession, err := createChatSession(ctx, store, model)
			if err != nil {
				fmt.Println("Unable to create new session:", err)
				continue
			}

			session = newSession
			history = []llm.Message{}

			fmt.Println("Started a new conversation.")
			fmt.Println("Session:", session.ID)
			continue

		case input == "/models":
			models, err := provider.Models(ctx)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			fmt.Println("\nAvailable models:")
			for i, name := range models {
				if name == model {
					fmt.Printf("  %d. %s (active)\n", i+1, name)
				} else {
					fmt.Printf("  %d. %s\n", i+1, name)
				}
			}
			continue

		case input == "/model":
			fmt.Println("Current model:", model)
			continue

		case strings.HasPrefix(input, "/model "):
			newModel := strings.TrimSpace(strings.TrimPrefix(input, "/model "))

			if newModel == "" {
				fmt.Println("Please provide a model name.")
				continue
			}

			models, err := provider.Models(ctx)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			found := false
			for _, name := range models {
				if name == newModel {
					found = true
					break
				}
			}

			if !found {
				fmt.Println("Model not found:", newModel)
				fmt.Println("Use /models to see available models.")
				continue
			}

			newSession, err := createChatSession(ctx, store, newModel)
			if err != nil {
				fmt.Println("Unable to create new session:", err)
				continue
			}

			model = newModel
			session = newSession
			history = []llm.Message{}

			fmt.Println("Switched model to:", model)
			fmt.Println("Started a new conversation.")
			fmt.Println("Session:", session.ID)
			continue

		case input == "/history":
			sessions, err := store.List(ctx, 20)
			if err != nil {
				fmt.Println("Unable to retrieve history:", err)
				continue
			}

			if len(sessions) == 0 {
				fmt.Println("No saved conversations.")
				continue
			}

			fmt.Println("\nRecent conversations:")
			for i, savedSession := range sessions {
				active := ""
				if savedSession.ID == session.ID {
					active = " (active)"
				}

				fmt.Printf(
					"\n%d. %s%s\n   ID: %s\n   Model: %s | Updated: %s\n",
					i+1,
					savedSession.Title,
					active,
					savedSession.ID,
					savedSession.Model,
					savedSession.UpdatedAt.Local().Format("02 Jan 2006 15:04"),
				)
			}
			continue

		case strings.HasPrefix(input, "/resume"):
			args := strings.Fields(input)

			if len(args) != 2 || args[0] != "/resume" {
				fmt.Println("Usage: /resume <session-id>")
				continue
			}

			savedSession, err := store.GetByID(ctx, args[1])
			if err != nil {
				fmt.Println("Unable to find conversation:", err)
				continue
			}

			if savedSession.Provider != "ollama" {
				fmt.Println("Unsupported provider:", savedSession.Provider)
				continue
			}

			savedHistory, err := loadChatHistory(ctx, store, savedSession.ID)
			if err != nil {
				fmt.Println("Unable to load conversation:", err)
				continue
			}

			session = savedSession
			model = savedSession.Model
			history = savedHistory

			fmt.Println("Resumed conversation:", session.Title)
			fmt.Println("Model:", model)
			fmt.Printf("Loaded %d messages.\n", len(history))
			continue

		case strings.HasPrefix(input, "/delete"):
			args := strings.Fields(input)

			if len(args) != 2 || args[0] != "/delete" {
				fmt.Println("Usage: /delete <session-id>")
				continue
			}

			if args[1] == session.ID {
				fmt.Println("Cannot delete the active conversation.")
				fmt.Println("Use /clear to start a new conversation first.")
				continue
			}

			if err := store.Delete(ctx, args[1]); err != nil {
				fmt.Println("Unable to delete conversation:", err)
				continue
			}

			fmt.Println("Conversation deleted.")
			continue
		}

		// Construct request using the current conversation history.
		requestHistory := append(
			append([]llm.Message{}, history...),
			llm.Message{
				Role:    "user",
				Content: input,
			},
		)

		request := llm.ChatRequest{
			Model:    model,
			Messages: requestHistory,
		}

		chatCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		fmt.Print("\nTermAI > ")

		response, err := provider.ChatStream(
			chatCtx,
			request,
			func(chunk string) {
				fmt.Print(chunk)
			},
		)

		cancel()
		fmt.Println()

		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

		// Prepare updated session metadata.
		updatedSession := *session

		if updatedSession.Title == "New conversation" {
			title := input
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			updatedSession.Title = title
		}

		userMessage := &conversation.Message{
			SessionID: session.ID,
			Role:      "user",
			Content:   input,
		}

		assistantMessage := &conversation.Message{
			SessionID: session.ID,
			Role:      "assistant",
			Content:   response.Message.Content,
		}

		// Save both messages and session metadata atomically.
		if err := store.SaveExchange(
			ctx,
			&updatedSession,
			userMessage,
			assistantMessage,
		); err != nil {
			fmt.Println("Warning: conversation exchange was not persisted:", err)
		} else {
			*session = updatedSession
		}

		// Keep in-memory history even if persistence fails.
		history = append(
			history,
			llm.Message{
				Role:    "user",
				Content: input,
			},
			response.Message,
		)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Input error:", err)
	}

	fmt.Println("\nGoodbye!")
}

func createChatSession(
	ctx context.Context,
	store storage.Store,
	model string,
) (*conversation.Session, error) {
	session := &conversation.Session{
		ID:       uuid.NewString(),
		Title:    "New conversation",
		Provider: "ollama",
		Model:    model,
	}

	if err := store.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create chat session: %w", err)
	}

	return session, nil
}

func loadChatHistory(
	ctx context.Context,
	store storage.Store,
	sessionID string,
) ([]llm.Message, error) {
	messages, err := store.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load messages: %w", err)
	}

	history := make([]llm.Message, 0, len(messages))

	for _, message := range messages {
		history = append(history, llm.Message{
			Role:    message.Role,
			Content: message.Content,
		})
	}

	return history, nil
}

func printChatCommands() {
	fmt.Println("\nCommands:")
	fmt.Println("  /models          List available models")
	fmt.Println("  /model           Show current model")
	fmt.Println("  /model <name>    Switch model")
	fmt.Println("  /history         List saved conversations")
	fmt.Println("  /resume <id>     Resume a conversation")
	fmt.Println("  /delete <id>     Delete a conversation")
	fmt.Println("  /clear           Start a new conversation")
	fmt.Println("  /exit            Exit TermAI")
}

func printUsage() {
	fmt.Print(`
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
