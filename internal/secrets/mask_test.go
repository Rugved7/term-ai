package secrets

import "testing"

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "typical Gemini key",
			input: "AIzaSyDaBcDeFgHiJkLmNoPqRsTuVwXyZ1234567x7Q2",
			want:  "AIza...x7Q2",
		},
		{
			name:  "short key",
			input: "abc123",
			want:  "******",
		},
		{
			name:  "empty key",
			input: "",
			want:  "",
		},
		{
			name:  "exactly 8 characters",
			input: "12345678",
			want:  "********",
		},
		{
			name:  "9 characters",
			input: "123456789",
			want:  "1234...6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskAPIKey(tt.input)
			if got != tt.want {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
