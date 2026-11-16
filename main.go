package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Rugved7/termai/internal/conversation"
	"github.com/Rugved7/termai/internal/llm"
	"github.com/Rugved7/termai/internal/storage"
)

// inputResult carries terminal input from the scanner goroutine.
type inputResult struct {
	text string
	err  error
	ok   bool
}

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
	)
	defer stop()

	rootCmd := newRootCmd()
	rootCmd.SetContext(ctx)

	if err := rootCmd.Execute(); err != nil {
		// Ctrl+C is an expected shutdown, not an application failure.
		if errors.Is(err, context.Canceled) || ctx.Err() != nil {
			return 0
		}

		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	return 0
}

// func startChat(ctx context.Context,provider llm.Provider,model string,store storage.Store,) error {
// 	scanner := bufio.NewScanner(os.Stdin)
// 	scanner.Buffer(make([]byte, 4096), 1024*1024)

// 	// Read terminal input in a separate goroutine so the main
// 	// REPL loop can respond to context cancellation.
// 	inputCh := make(chan inputResult, 1)

// 	go func() {
// 		for scanner.Scan() {
// 			inputCh <- inputResult{
// 				text: scanner.Text(),
// 				ok:   true,
// 			}
// 		}

// 		inputCh <- inputResult{
// 			err: scanner.Err(),
// 			ok:  false,
// 		}
// 	}()

// 	session, err := createChatSession(ctx, store, model)
// 	if err != nil {
// 		return fmt.Errorf("initialize chat session: %w", err)
// 	}

// 	history := []llm.Message{}

// 	fmt.Print(`
//  _____                   ___
// |_   _|__ _ __ _ __ ___ |_ _|
//   | |/ _ \ '__| '_ ' _ \ | |
//   | |  __/ |  | | | | || |
//   |_|\___|_|  |_| |_| |_|___|

// Welcome to TermAI
// `)

// 	fmt.Println("Provider :", "Ollama")
// 	fmt.Println("Model    :", model)
// 	fmt.Println("Session  :", session.ID)

// 	printChatCommands()

// 	for {
// 		if ctx.Err() != nil {
// 			fmt.Println("\nGoodbye!")
// 			return nil
// 		}

// 		fmt.Print("\nYou > ")

// 		var result inputResult

// 		select {
// 		case <-ctx.Done():
// 			fmt.Println("\nGoodbye!")
// 			return nil

// 		case result = <-inputCh:
// 		}

// 		if !result.ok {
// 			if result.err != nil {
// 				return fmt.Errorf("read interactive input: %w", result.err)
// 			}

// 			fmt.Println("\nGoodbye!")
// 			return nil
// 		}

// 		input := strings.TrimSpace(result.text)
// 		if input == "" {
// 			continue
// 		}

// 		switch {
// 		case input == "/exit":
// 			fmt.Println("Goodbye!")
// 			return nil

// 		case input == "/clear":
// 			newSession, err := createChatSession(ctx, store, model)
// 			if err != nil {
// 				if ctx.Err() != nil {
// 					fmt.Println("\nGoodbye!")
// 					return nil
// 				}

// 				fmt.Println("Unable to create new session:", err)
// 				continue
// 			}

// 			session = newSession
// 			history = []llm.Message{}

// 			fmt.Println("Started a new conversation.")
// 			fmt.Println("Session:", session.ID)
// 			continue

// 		case input == "/models":
// 			models, err := provider.Models(ctx)
// 			if err != nil {
// 				if ctx.Err() != nil {
// 					fmt.Println("\nGoodbye!")
// 					return nil
// 				}

// 				fmt.Println("Error:", formatOllamaError("list models", err))
// 				continue
// 			}

// 			fmt.Println("\nAvailable models:")

// 			for i, name := range models {
// 				if name == model {
// 					fmt.Printf("  %d. %s (active)\n", i+1, name)
// 				} else {
// 					fmt.Printf("  %d. %s\n", i+1, name)
// 				}
// 			}

// 			continue

// 		case input == "/model":
// 			fmt.Println("Current model:", model)
// 			continue

// 		case strings.HasPrefix(input, "/model "):
// 			newModel := strings.TrimSpace(
// 				strings.TrimPrefix(input, "/model "),
// 			)

// 			if newModel == "" {
// 				fmt.Println("Please provide a model name.")
// 				continue
// 			}

// 			models, err := provider.Models(ctx)
// 			if err != nil {
// 				if ctx.Err() != nil {
// 					fmt.Println("\nGoodbye!")
// 					return nil
// 				}

// 				fmt.Println("Error:", formatOllamaError("list models", err))
// 				continue
// 			}

// 			found := false

// 			for _, name := range models {
// 				if name == newModel {
// 					found = true
// 					break
// 				}
// 			}

// 			if !found {
// 				fmt.Println("Model not found:", newModel)
// 				fmt.Println("Use /models to see available models.")
// 				continue
// 			}

// 			newSession, err := createChatSession(ctx, store, newModel)
// 			if err != nil {
// 				if ctx.Err() != nil {
// 					fmt.Println("\nGoodbye!")
// 					return nil
// 				}

// 				fmt.Println("Unable to create new session:", err)
// 				continue
// 			}

// 			model = newModel
// 			session = newSession
// 			history = []llm.Message{}

// 			fmt.Println("Switched model to:", model)
// 			fmt.Println("Started a new conversation.")
// 			fmt.Println("Session:", session.ID)
// 			continue

// 		case input == "/history":
// 			sessions, err := store.List(ctx, 20)
// 			if err != nil {
// 				fmt.Println("Unable to retrieve history:", err)
// 				continue
// 			}

// 			if len(sessions) == 0 {
// 				fmt.Println("No saved conversations.")
// 				continue
// 			}

// 			fmt.Println("\nRecent conversations:")

// 			for i, savedSession := range sessions {
// 				active := ""

// 				if savedSession.ID == session.ID {
// 					active = " (active)"
// 				}

// 				fmt.Printf(
// 					"\n%d. %s%s\n   ID: %s\n   Model: %s | Updated: %s\n",
// 					i+1,
// 					savedSession.Title,
// 					active,
// 					savedSession.ID,
// 					savedSession.Model,
// 					savedSession.UpdatedAt.Local().Format("02 Jan 2006 15:04"),
// 				)
// 			}

// 			continue

// 		case strings.HasPrefix(input, "/resume"):
// 			args := strings.Fields(input)

// 			if len(args) != 2 || args[0] != "/resume" {
// 				fmt.Println("Usage: /resume <session-id>")
// 				continue
// 			}

// 			savedSession, err := store.GetByID(ctx, args[1])
// 			if err != nil {
// 				fmt.Println("Unable to find conversation:", err)
// 				continue
// 			}

// 			if savedSession.Provider != "ollama" {
// 				fmt.Println("Unsupported provider:", savedSession.Provider)
// 				continue
// 			}

// 			savedHistory, err := loadChatHistory(
// 				ctx,
// 				store,
// 				savedSession.ID,
// 			)
// 			if err != nil {
// 				fmt.Println("Unable to load conversation:", err)
// 				continue
// 			}

// 			session = savedSession
// 			model = savedSession.Model
// 			history = savedHistory

// 			fmt.Println("Resumed conversation:", session.Title)
// 			fmt.Println("Model:", model)
// 			fmt.Printf("Loaded %d messages.\n", len(history))
// 			continue

// 		case strings.HasPrefix(input, "/delete"):
// 			args := strings.Fields(input)

// 			if len(args) != 2 || args[0] != "/delete" {
// 				fmt.Println("Usage: /delete <session-id>")
// 				continue
// 			}

// 			if args[1] == session.ID {
// 				fmt.Println("Cannot delete the active conversation.")
// 				fmt.Println("Use /clear to start a new conversation first.")
// 				continue
// 			}

// 			if err := store.Delete(ctx, args[1]); err != nil {
// 				fmt.Println("Unable to delete conversation:", err)
// 				continue
// 			}

// 			fmt.Println("Conversation deleted.")
// 			continue
// 		}

// 		// Construct request using the current conversation history.
// 		requestHistory := append(
// 			append([]llm.Message{}, history...),
// 			llm.Message{
// 				Role:    "user",
// 				Content: input,
// 			},
// 		)

// 		request := llm.ChatRequest{
// 			Model:    model,
// 			Messages: requestHistory,
// 		}

// 		chatCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

// 		fmt.Print("\nTermAI > ")

// 		response, err := provider.ChatStream(
// 			chatCtx,
// 			request,
// 			func(chunk string) {
// 				fmt.Print(chunk)
// 			},
// 		)

// 		cancel()
// 		fmt.Println()

// 		if err != nil {
// 			if ctx.Err() != nil {
// 				fmt.Println("Goodbye!")
// 				return nil
// 			}

// 			fmt.Println("Error:", formatOllamaError("generate response", err))
// 			continue
// 		}

// 		// Prepare updated session metadata.
// 		updatedSession := *session

// 		if updatedSession.Title == "New conversation" {
// 			title := input

// 			if len(title) > 60 {
// 				title = title[:57] + "..."
// 			}

// 			updatedSession.Title = title
// 		}

// 		userMessage := &conversation.Message{
// 			SessionID: session.ID,
// 			Role:      "user",
// 			Content:   input,
// 		}

// 		assistantMessage := &conversation.Message{
// 			SessionID: session.ID,
// 			Role:      "assistant",
// 			Content:   response.Message.Content,
// 		}

// 		// Save both messages and session metadata atomically.
// 		if err := store.SaveExchange(
// 			ctx,
// 			&updatedSession,
// 			userMessage,
// 			assistantMessage,
// 		); err != nil {
// 			if ctx.Err() != nil {
// 				fmt.Println("Goodbye!")
// 				return nil
// 			}

// 			fmt.Println(
// 				"Warning: conversation exchange was not persisted:",
// 				err,
// 			)
// 		} else {
// 			*session = updatedSession
// 		}

// 		// Keep in-memory history even if persistence fails.
// 		history = append(
// 			history,
// 			llm.Message{
// 				Role:    "user",
// 				Content: input,
// 			},
// 			response.Message,
// 		)
// 	}
// }

func startChat(
	ctx context.Context,
	provider llm.Provider,
	model string,
	store storage.Store,
) error {
	return startChatWithIO(
		ctx,
		provider,
		model,
		store,
		os.Stdin,
		os.Stdout,
	)
}

func startChatWithIO(
	ctx context.Context,
	provider llm.Provider,
	model string,
	store storage.Store,
	input io.Reader,
	output io.Writer,
) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	inputCh := make(chan inputResult, 1)

	done := make(chan struct{})
	defer close(done)

	go func() {
		for scanner.Scan() {
			select {
			case inputCh <- inputResult{
				text: scanner.Text(),
				ok:   true,
			}:
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}

		select {
		case inputCh <- inputResult{
			err: scanner.Err(),
			ok:  false,
		}:
		case <-ctx.Done():
		case <-done:
		}
	}()

	session, err := createChatSession(ctx, store, model)
	if err != nil {
		return fmt.Errorf("initialize chat session: %w", err)
	}

	history := []llm.Message{}
	fmt.Fprint(output, `
 _____                      _    ___
|_   _|__ _ __ _ __ ___    / \  |_ _|
  | |/ _ \ '__| '_ ' _ \  / _ \  | |
  | |  __/ |  | | | | | |/ ___ \ | |
  |_|\___|_|  |_| |_| |_/_/   \_\___|

Welcome to TermAI
`)

	fmt.Fprintln(output, "Provider :", "Ollama")
	fmt.Fprintln(output, "Model    :", model)
	fmt.Fprintln(output, "Session  :", session.ID)

	printChatCommandsTo(output)

	for {
		if ctx.Err() != nil {
			fmt.Fprintln(output, "\nGoodbye!")
			return nil
		}

		fmt.Fprint(output, "\nYou > ")

		var result inputResult

		select {
		case <-ctx.Done():
			fmt.Fprintln(output, "\nGoodbye!")
			return nil

		case result = <-inputCh:
		}

		if !result.ok {
			if result.err != nil {
				return fmt.Errorf("read interactive input: %w", result.err)
			}

			fmt.Fprintln(output, "\nGoodbye!")
			return nil
		}

		inputText := strings.TrimSpace(result.text)
		if inputText == "" {
			continue
		}

		switch {
		case inputText == "/exit":
			fmt.Fprintln(output, "Goodbye!")
			return nil

		case inputText == "/clear":
			newSession, err := createChatSession(ctx, store, model)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Fprintln(output, "\nGoodbye!")
					return nil
				}

				fmt.Fprintln(output, "Unable to create new session:", err)
				continue
			}

			session = newSession
			history = []llm.Message{}

			fmt.Fprintln(output, "Started a new conversation.")
			fmt.Fprintln(output, "Session:", session.ID)
			continue

		case inputText == "/models":
			models, err := provider.Models(ctx)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Fprintln(output, "\nGoodbye!")
					return nil
				}

				fmt.Fprintln(output, "Error:", formatOllamaError("list models", err))
				continue
			}

			fmt.Fprintln(output, "\nAvailable models:")

			for i, name := range models {
				if name == model {
					fmt.Fprintf(output, "  %d. %s (active)\n", i+1, name)
				} else {
					fmt.Fprintf(output, "  %d. %s\n", i+1, name)
				}
			}

			continue

		case inputText == "/model":
			fmt.Fprintln(output, "Current model:", model)
			continue

		case strings.HasPrefix(inputText, "/model "):
			newModel := strings.TrimSpace(
				strings.TrimPrefix(inputText, "/model "),
			)

			if newModel == "" {
				fmt.Fprintln(output, "Please provide a model name.")
				continue
			}

			models, err := provider.Models(ctx)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Fprintln(output, "\nGoodbye!")
					return nil
				}

				fmt.Fprintln(output, "Error:", formatOllamaError("list models", err))
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
				fmt.Fprintln(output, "Model not found:", newModel)
				fmt.Fprintln(output, "Use /models to see available models.")
				continue
			}

			newSession, err := createChatSession(ctx, store, newModel)
			if err != nil {
				if ctx.Err() != nil {
					fmt.Fprintln(output, "\nGoodbye!")
					return nil
				}

				fmt.Fprintln(output, "Unable to create new session:", err)
				continue
			}

			model = newModel
			session = newSession
			history = []llm.Message{}

			fmt.Fprintln(output, "Switched model to:", model)
			fmt.Fprintln(output, "Started a new conversation.")
			fmt.Fprintln(output, "Session:", session.ID)
			continue

		case inputText == "/history":
			sessions, err := store.List(ctx, 20)
			if err != nil {
				fmt.Fprintln(output, "Unable to retrieve history:", err)
				continue
			}

			if len(sessions) == 0 {
				fmt.Fprintln(output, "No saved conversations.")
				continue
			}

			fmt.Fprintln(output, "\nRecent conversations:")

			for i, savedSession := range sessions {
				active := ""
				if savedSession.ID == session.ID {
					active = " (active)"
				}

				fmt.Fprintf(
					output,
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

		case strings.HasPrefix(inputText, "/resume"):
			args := strings.Fields(inputText)

			if len(args) != 2 || args[0] != "/resume" {
				fmt.Fprintln(output, "Usage: /resume <session-id>")
				continue
			}

			savedSession, err := store.GetByID(ctx, args[1])
			if err != nil {
				fmt.Fprintln(output, "Unable to find conversation:", err)
				continue
			}

			if savedSession.Provider != "ollama" {
				fmt.Fprintln(output, "Unsupported provider:", savedSession.Provider)
				continue
			}

			savedHistory, err := loadChatHistory(ctx, store, savedSession.ID)
			if err != nil {
				fmt.Fprintln(output, "Unable to load conversation:", err)
				continue
			}

			session = savedSession
			model = savedSession.Model
			history = savedHistory

			fmt.Fprintln(output, "Resumed conversation:", session.Title)
			fmt.Fprintln(output, "Model:", model)
			fmt.Fprintf(output, "Loaded %d messages.\n", len(history))
			continue

		case strings.HasPrefix(inputText, "/delete"):
			args := strings.Fields(inputText)

			if len(args) != 2 || args[0] != "/delete" {
				fmt.Fprintln(output, "Usage: /delete <session-id>")
				continue
			}

			if args[1] == session.ID {
				fmt.Fprintln(output, "Cannot delete the active conversation.")
				fmt.Fprintln(output, "Use /clear to start a new conversation first.")
				continue
			}

			if err := store.Delete(ctx, args[1]); err != nil {
				fmt.Fprintln(output, "Unable to delete conversation:", err)
				continue
			}

			fmt.Fprintln(output, "Conversation deleted.")
			continue
		}

		requestHistory := append(
			append([]llm.Message{}, history...),
			llm.Message{
				Role:    "user",
				Content: inputText,
			},
		)

		request := llm.ChatRequest{
			Model:    model,
			Messages: requestHistory,
		}

		chatCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)

		fmt.Fprint(output, "\nTermAI > ")

		response, err := provider.ChatStream(
			chatCtx,
			request,
			func(chunk string) {
				fmt.Fprint(output, chunk)
			},
		)

		cancel()
		fmt.Fprintln(output)

		if err != nil {
			if ctx.Err() != nil {
				fmt.Fprintln(output, "Goodbye!")
				return nil
			}

			fmt.Fprintln(output, "Error:", formatOllamaError("generate response", err))
			continue
		}

		updatedSession := *session

		if updatedSession.Title == "New conversation" {
			title := inputText
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			updatedSession.Title = title
		}

		userMessage := &conversation.Message{
			SessionID: session.ID,
			Role:      "user",
			Content:   inputText,
		}

		assistantMessage := &conversation.Message{
			SessionID: session.ID,
			Role:      "assistant",
			Content:   response.Message.Content,
		}

		if err := store.SaveExchange(
			ctx,
			&updatedSession,
			userMessage,
			assistantMessage,
		); err != nil {
			if ctx.Err() != nil {
				fmt.Fprintln(output, "Goodbye!")
				return nil
			}

			fmt.Fprintln(
				output,
				"Warning: conversation exchange was not persisted:",
				err,
			)
		} else {
			*session = updatedSession
		}

		history = append(
			history,
			llm.Message{
				Role:    "user",
				Content: inputText,
			},
			response.Message,
		)
	}
}

func printChatCommandsTo(output io.Writer) {
	fmt.Fprintln(output, "\nCommands:")
	fmt.Fprintln(output, "  /models          List available models")
	fmt.Fprintln(output, "  /model           Show current model")
	fmt.Fprintln(output, "  /model <name>    Switch model")
	fmt.Fprintln(output, "  /history         List saved conversations")
	fmt.Fprintln(output, "  /resume <id>     Resume a conversation")
	fmt.Fprintln(output, "  /delete <id>     Delete a conversation")
	fmt.Fprintln(output, "  /clear           Start a new conversation")
	fmt.Fprintln(output, "  /exit            Exit TermAI")
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
