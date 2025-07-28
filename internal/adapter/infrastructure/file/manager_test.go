//go:build unit

package file

import (
"os"
"path/filepath"
"testing"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestNewManagerAdapter(t *testing.T) {
adapter := NewManagerAdapter()
assert.NotNil(t, adapter)
}

func TestManagerAdapter_WriteAndReadFile(t *testing.T) {
adapter := NewManagerAdapter()

// Create a temporary directory for testing
tempDir := t.TempDir()
testFile := filepath.Join(tempDir, "test.txt")
testContent := []byte("test content")

t.Run("WriteFile", func(t *testing.T) {
err := adapter.WriteFile(testFile, testContent, 0644)
assert.NoError(t, err)

// Verify file was created with correct permissions
info, err := os.Stat(testFile)
require.NoError(t, err)
assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
})

t.Run("ReadFile", func(t *testing.T) {
content, err := adapter.ReadFile(testFile)
assert.NoError(t, err)
assert.Equal(t, testContent, content)
})

t.Run("FileExists", func(t *testing.T) {
exists := adapter.FileExists(testFile)
assert.True(t, exists)

nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
exists = adapter.FileExists(nonExistentFile)
assert.False(t, exists)
})
}

func TestManagerAdapter_ReadFile_NonExistent(t *testing.T) {
adapter := NewManagerAdapter()

_, err := adapter.ReadFile("/nonexistent/file.txt")
assert.Error(t, err)
assert.Contains(t, err.Error(), "failed to read file")
}

func TestManagerAdapter_WriteFile_InvalidPath(t *testing.T) {
adapter := NewManagerAdapter()

// Try to write to a directory that doesn't exist and can't be created
err := adapter.WriteFile("/nonexistent/directory/file.txt", []byte("test"), 0644)
assert.Error(t, err)
assert.Contains(t, err.Error(), "failed to write file")
}
