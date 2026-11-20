package terminal

import (
	"strings"
	"testing"
)

func TestReadPipedInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "reads simple text",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "reads multiline text",
			input: "line 1\nline 2\nline 3",
			want:  "line 1\nline 2\nline 3",
		},
		{
			name:  "reads empty input",
			input: "",
			want:  "",
		},
		{
			name:  "preserves whitespace",
			input: "  spaces  \n\ttabs\t\n",
			want:  "  spaces  \n\ttabs\t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			got, err := ReadPipedInput(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadPipedInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ReadPipedInput() = %q, want %q", got, tt.want)
			}
		})
	}
}
