package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Rugved7/termai/internal/llm"
	"github.com/Rugved7/termai/internal/storage"
	"github.com/Rugved7/termai/internal/storage/sqlite"
)

// mockProvider simulates an LLM provider for integration tests.
type mockProvider struct {
	response      string
	responses     []string
	streamErrors  []error
	responseIndex int
	request       llm.ChatRequest
	requests      []llm.ChatRequest
}

func (m *mockProvider) Chat(
	ctx context.Context,
	req llm.ChatRequest,
) (llm.ChatResponse, error) {
	m.request = req
	m.requests = append(m.requests, req)

	return llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: m.response,
		},
	}, nil
}

func (m *mockProvider) ChatStream(
	ctx context.Context,
	req llm.ChatRequest,
	onChunk func(string),
) (llm.ChatResponse, error) {
	m.request = req
	m.requests = append(m.requests, req)

	index := m.responseIndex
	m.responseIndex++

	if index < len(m.streamErrors) && m.streamErrors[index] != nil {
		return llm.ChatResponse{}, m.streamErrors[index]
	}

	response := m.response

	if index < len(m.responses) {
		response = m.responses[index]
	}

	onChunk(response)

	return llm.ChatResponse{
		Message: llm.Message{
			Role:    "assistant",
			Content: response,
		},
	}, nil
}

func (m *mockProvider) Models(
	ctx context.Context,
) ([]string, error) {
	return []string{"test-model"}, nil
}

// newTestStore creates an isolated SQLite database.
func newTestStore(t *testing.T) storage.Store {
	store, err := sqlite.New(t.TempDir())
	if err != nil {
		t.Fatalf("initialize SQLite store: %v", err)
	}

	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close SQLite store: %v", err)
		}
	})

	return store
}

// Test 1: Verify user and assistant messages are persisted.
func TestInteractiveChatPersistsExchange(t *testing.T) {
	ctx := context.Background()

	store := newTestStore(t)

	provider := &mockProvider{
		response: "Hello from mock provider!",
	}

	input := strings.NewReader(
		"Hello TermAI\n/exit\n",
	)

	var output bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		input,
		&output,
	)
	if err != nil {
		t.Fatalf("start interactive chat: %v", err)
	}

	if !strings.Contains(output.String(), provider.response) {
		t.Errorf("output does not contain assistant response")
	}

	if provider.request.Model != "test-model" {
		t.Errorf(
			"model = %q, want test-model",
			provider.request.Model,
		)
	}

	if len(provider.request.Messages) != 1 {
		t.Fatalf(
			"request message count = %d, want 1",
			len(provider.request.Messages),
		)
	}

	if provider.request.Messages[0].Content != "Hello TermAI" {
		t.Errorf("unexpected user prompt: %+v", provider.request.Messages[0])
	}

	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}

	session := sessions[0]

	if session.Title != "Hello TermAI" {
		t.Errorf(
			"session title = %q, want Hello TermAI",
			session.Title,
		)
	}

	messages, err := store.ListBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}

	if messages[0].Role != "user" ||
		messages[0].Content != "Hello TermAI" {
		t.Errorf("unexpected user message: %+v", messages[0])
	}

	if messages[1].Role != "assistant" ||
		messages[1].Content != provider.response {
		t.Errorf("unexpected assistant message: %+v", messages[1])
	}
}

// Test 2: Verify conversation history is included in subsequent requests.
func TestInteractiveChatMaintainsConversationHistory(t *testing.T) {
	ctx := context.Background()

	store := newTestStore(t)

	provider := &mockProvider{
		response: "Hi there!",
	}

	input := strings.NewReader(
		"Hello\nHow are you?\n/exit\n",
	)

	var output bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		input,
		&output,
	)
	if err != nil {
		t.Fatalf("start interactive chat: %v", err)
	}

	if len(provider.requests) != 2 {
		t.Fatalf(
			"provider request count = %d, want 2",
			len(provider.requests),
		)
	}

	firstRequest := provider.requests[0]

	if len(firstRequest.Messages) != 1 {
		t.Fatalf(
			"first request message count = %d, want 1",
			len(firstRequest.Messages),
		)
	}

	secondRequest := provider.requests[1]

	expected := []llm.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	if len(secondRequest.Messages) != len(expected) {
		t.Fatalf(
			"second request message count = %d, want %d",
			len(secondRequest.Messages),
			len(expected),
		)
	}

	for i, want := range expected {
		got := secondRequest.Messages[i]

		if got.Role != want.Role || got.Content != want.Content {
			t.Errorf(
				"message[%d] = %+v, want %+v",
				i,
				got,
				want,
			)
		}
	}

	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}

	messages, err := store.ListBySession(ctx, sessions[0].ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(messages) != 4 {
		t.Fatalf("persisted message count = %d, want 4", len(messages))
	}
}

// Test 3: Verify conversations can be resumed across REPL executions.
func TestInteractiveChatResumesConversation(t *testing.T) {
	ctx := context.Background()

	store := newTestStore(t)

	provider := &mockProvider{
		responses: []string{
			"Hi there!",
			"I'm doing well, thanks!",
		},
	}

	// First REPL execution.
	firstInput := strings.NewReader(
		"Hello\n/exit\n",
	)

	var firstOutput bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		firstInput,
		&firstOutput,
	)
	if err != nil {
		t.Fatalf("first chat execution: %v", err)
	}

	// Retrieve the saved session ID.
	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}

	sessionID := sessions[0].ID

	// Second REPL execution using the same database.
	secondInput := strings.NewReader(
		"/history\n/resume " + sessionID +
			"\nHow are you?\n/exit\n",
	)

	var secondOutput bytes.Buffer

	err = startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		secondInput,
		&secondOutput,
	)
	if err != nil {
		t.Fatalf("second chat execution: %v", err)
	}

	output := secondOutput.String()

	if !strings.Contains(output, "Recent conversations:") {
		t.Error("history command did not list saved conversations")
	}

	if !strings.Contains(output, "Resumed conversation: Hello") {
		t.Error("conversation was not resumed")
	}

	if !strings.Contains(output, "Loaded 2 messages.") {
		t.Error("previous conversation messages were not restored")
	}

	// Verify the second execution sent the restored context.
	if len(provider.requests) != 2 {
		t.Fatalf(
			"provider request count = %d, want 2",
			len(provider.requests),
		)
	}

	resumedRequest := provider.requests[1]

	expected := []llm.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	if len(resumedRequest.Messages) != len(expected) {
		t.Fatalf(
			"resumed request message count = %d, want %d",
			len(resumedRequest.Messages),
			len(expected),
		)
	}

	for i, want := range expected {
		got := resumedRequest.Messages[i]

		if got.Role != want.Role || got.Content != want.Content {
			t.Errorf(
				"resumed message[%d] = %+v, want %+v",
				i,
				got,
				want,
			)
		}
	}

	// Verify both executions persisted messages in the same session.
	messages, err := store.ListBySession(ctx, sessionID)
	if err != nil {
		t.Fatalf("list resumed session messages: %v", err)
	}

	if len(messages) != 4 {
		t.Fatalf(
			"persisted message count = %d, want 4",
			len(messages),
		)
	}
}

func TestInteractiveChatRecoversFromProviderError(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	provider := &mockProvider{
		streamErrors: []error{
			errors.New("mock provider failure"),
		},
		responses: []string{
			"",
			"Recovered successfully!",
		},
	}
	input := strings.NewReader(
		"First prompt\nSecond prompt\n/exit\n",
	)

	var output bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		input,
		&output,
	)
	if err != nil {
		t.Fatalf("start interactive chat: %v", err)
	}

	if !strings.Contains(output.String(), "mock provider failure") {
		t.Error("expected provider error in terminal output")
	}

	if !strings.Contains(output.String(), "Recovered successfully!") {
		t.Error("REPL did not recover after provider failure")
	}

	if len(provider.requests) != 2 {
		t.Fatalf(
			"provider request count = %d, want 2",
			len(provider.requests),
		)
	}

	// Failed prompts should not enter conversation history.
	secondRequest := provider.requests[1]

	if len(secondRequest.Messages) != 1 {
		t.Fatalf(
			"second request message count = %d, want 1",
			len(secondRequest.Messages),
		)
	}

	if secondRequest.Messages[0].Content != "Second prompt" {
		t.Errorf(
			"second prompt = %q, want Second prompt",
			secondRequest.Messages[0].Content,
		)
	}

	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}

	messages, err := store.ListBySession(ctx, sessions[0].ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("persisted message count = %d, want 2", len(messages))
	}
}

func TestInteractiveChatIgnoresEmptyInput(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	provider := &mockProvider{
		response: "Unexpected response",
	}

	input := strings.NewReader(
		"\n   \n\t\n/exit\n",
	)

	var output bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		input,
		&output,
	)
	if err != nil {
		t.Fatalf("start interactive chat: %v", err)
	}

	if len(provider.requests) != 0 {
		t.Fatalf(
			"provider received %d requests, want 0",
			len(provider.requests),
		)
	}

	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}

	messages, err := store.ListBySession(ctx, sessions[0].ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(messages) != 0 {
		t.Fatalf("message count = %d, want 0", len(messages))
	}
}

func TestInteractiveChatClearStartsNewSession(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	provider := &mockProvider{
		responses: []string{
			"First response",
			"Second response",
		},
	}

	input := strings.NewReader(
		"Hello\n/clear\nAfter clear\n/exit\n",
	)

	var output bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		input,
		&output,
	)
	if err != nil {
		t.Fatalf("start interactive chat: %v", err)
	}

	if !strings.Contains(output.String(), "Started a new conversation.") {
		t.Error("expected new conversation confirmation")
	}

	if len(provider.requests) != 2 {
		t.Fatalf(
			"provider request count = %d, want 2",
			len(provider.requests),
		)
	}

	secondRequest := provider.requests[1]

	if len(secondRequest.Messages) != 1 {
		t.Fatalf(
			"second request message count = %d, want 1",
			len(secondRequest.Messages),
		)
	}

	if secondRequest.Messages[0].Content != "After clear" {
		t.Errorf(
			"second prompt = %q, want After clear",
			secondRequest.Messages[0].Content,
		)
	}

	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("session count = %d, want 2", len(sessions))
	}

	// Verify that the original session still contains its messages.
	var totalMessages int

	for _, session := range sessions {
		messages, err := store.ListBySession(ctx, session.ID)
		if err != nil {
			t.Fatalf("list messages: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf(
				"session %s has %d messages, want 2",
				session.ID,
				len(messages),
			)
		}

		totalMessages += len(messages)
	}

	if totalMessages != 4 {
		t.Errorf("total persisted messages = %d, want 4", totalMessages)
	}
}

func TestInteractiveChatHandlesInvalidResume(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t)

	provider := &mockProvider{
		response: "Chat continued successfully!",
	}

	input := strings.NewReader(
		"/resume missing-session-id\nHello\n/exit\n",
	)

	var output bytes.Buffer

	err := startChatWithIO(
		ctx,
		provider,
		"test-model",
		store,
		input,
		&output,
	)
	if err != nil {
		t.Fatalf("start interactive chat: %v", err)
	}

	if !strings.Contains(output.String(), "Unable to find conversation:") {
		t.Error("expected invalid resume error")
	}

	if !strings.Contains(output.String(), "Chat continued successfully!") {
		t.Error("REPL did not continue after invalid resume")
	}

	if len(provider.requests) != 1 {
		t.Fatalf(
			"provider request count = %d, want 1",
			len(provider.requests),
		)
	}

	sessions, err := store.List(ctx, 20)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("session count = %d, want 1", len(sessions))
	}

	messages, err := store.ListBySession(ctx, sessions[0].ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("message count = %d, want 2", len(messages))
	}
}
