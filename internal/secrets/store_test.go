package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_SetAndGet_File(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	provider := "test-provider"
	apiKey := "test-api-key-12345"

	// Set credential
	err := store.setFile(provider, apiKey)
	if err != nil {
		t.Fatalf("setFile failed: %v", err)
	}

	// Get credential
	got, err := store.getFile(provider)
	if err != nil {
		t.Fatalf("getFile failed: %v", err)
	}

	if got != apiKey {
		t.Errorf("getFile() = %q, want %q", got, apiKey)
	}

	// Verify file permissions
	filePath := filepath.Join(tempDir, provider)
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}

	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("file permissions = %o, want 0600", mode.Perm())
	}
}

func TestStore_Delete_File(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	provider := "test-provider"
	apiKey := "test-api-key"

	// Set and verify
	if err := store.setFile(provider, apiKey); err != nil {
		t.Fatalf("setFile failed: %v", err)
	}

	// Delete
	if err := store.deleteFile(provider); err != nil {
		t.Fatalf("deleteFile failed: %v", err)
	}

	// Verify deleted
	_, err := store.getFile(provider)
	if err != ErrNotFound {
		t.Errorf("getFile after delete: got error %v, want ErrNotFound", err)
	}

	// Deleting again should not error
	if err := store.deleteFile(provider); err != nil {
		t.Errorf("deleteFile on non-existent file: %v", err)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	_, err := store.getFile("nonexistent")
	if err != ErrNotFound {
		t.Errorf("getFile for nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestStore_SetEmptyProvider(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	err := store.Set("", "some-key")
	if err == nil {
		t.Error("Set with empty provider should fail")
	}
}

func TestStore_SetEmptyKey(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	err := store.Set("provider", "")
	if err == nil {
		t.Error("Set with empty key should fail")
	}
}

func TestStore_Has(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	provider := "test-provider"

	// Should not have credential initially
	if store.Has(provider) {
		t.Error("Has() should return false for non-existent credential")
	}

	// Set credential via file
	if err := store.setFile(provider, "test-key"); err != nil {
		t.Fatalf("setFile failed: %v", err)
	}

	// Should now have credential
	if !store.Has(provider) {
		t.Error("Has() should return true after setting credential")
	}
}

func TestEnvVarName(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"gemini", "GEMINI_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"anthropic", "ANTHROPIC_API_KEY"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := envVarName(tt.provider)
			if got != tt.want {
				t.Errorf("envVarName(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestStore_GetPriority(t *testing.T) {
	tempDir := t.TempDir()
	store := NewStore(tempDir)

	provider := "gemini"
	fileKey := "file-stored-key"
	envKey := "env-override-key"

	// Store in file
	if err := store.setFile(provider, fileKey); err != nil {
		t.Fatalf("setFile failed: %v", err)
	}

	// Without env var, should get file key
	got, err := store.Get(provider)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != fileKey {
		t.Errorf("Get() = %q, want %q (from file)", got, fileKey)
	}

	// Set env var
	t.Setenv("GEMINI_API_KEY", envKey)

	// With env var, should get env key (higher priority)
	got, err = store.Get(provider)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != envKey {
		t.Errorf("Get() = %q, want %q (from env)", got, envKey)
	}
}
