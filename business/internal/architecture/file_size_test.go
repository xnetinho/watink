package architecture

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileSize_ChannelAndMessagePackagesUnder500Lines enforces that no production Go file
// in the channel and message packages exceeds 500 lines (Tarefa 5.5).
func TestFileSize_ChannelAndMessagePackagesUnder500Lines(t *testing.T) {
	root := infraRoot(t)

	targets := []string{
		filepath.Join(root, "business", "internal", "channels"),
		filepath.Join(root, "engine-go", "internal", "whatsapp"),
	}

	// Specific message controller prefixes in business/internal/controllers
	ctrlDir := filepath.Join(root, "business", "internal", "controllers")

	var violations []string

	checkFile := func(path string) {
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		lines := bytes.Count(data, []byte("\n")) + 1
		if lines > 500 {
			rel, _ := filepath.Rel(root, path)
			violations = append(violations, rel+" ("+itoa(lines)+" lines)")
		}
	}

	for _, dir := range targets {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				checkFile(path)
			}
			return nil
		})
		require.NoError(t, err)
	}

	// Controllers for message and quick answers
	entries, err := os.ReadDir(ctrlDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "message") || strings.HasPrefix(name, "quick_answer") {
			checkFile(filepath.Join(ctrlDir, name))
		}
	}

	assert.Empty(t, violations, "arquivos nos pacotes de canal e mensagem que excedem o limite de 500 linhas")
}
