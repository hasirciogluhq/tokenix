package input

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ResolveInput picks the input source with this precedence:
// 1) --data string
// 2) --file path
// 3) piped stdin
func ResolveInput(stdin io.Reader, data, filePath string) (string, string, error) {
	if strings.TrimSpace(data) != "" {
		return data, "flag:data", nil
	}

	if strings.TrimSpace(filePath) != "" {
		b, err := os.ReadFile(filePath)
		if err != nil {
			return "", "", fmt.Errorf("file could not be read: %w", err)
		}

		content := string(b)
		if strings.TrimSpace(content) == "" {
			return "", "", errors.New("file content is empty")
		}
		return content, "file:" + filePath, nil
	}

	if isStdinPiped(stdin) {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return "", "", fmt.Errorf("stdin could not be read: %w", err)
		}
		content := string(b)
		if strings.TrimSpace(content) == "" {
			return "", "", errors.New("stdin content is empty")
		}
		return content, "stdin", nil
	}

	return "", "", errors.New("no input found: provide --data, --file, or stdin")
}

func isStdinPiped(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return true
	}

	stat, err := file.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) == 0
}
