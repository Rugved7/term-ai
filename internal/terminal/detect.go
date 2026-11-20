package terminal

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IsPiped returns true if stdin is being piped from another command.
func IsPiped() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

// IsInteractive returns true if both stdin and stdout are terminals.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd()))
}

// ReadPipedInput reads all input from a reader (typically stdin).
// Returns empty string and no error if the reader is empty.
func ReadPipedInput(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
