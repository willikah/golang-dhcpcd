// Package file provides file system operations adapter implementation.
package file

import (
	"fmt"
	"os"

	"golang-dhcpcd/internal/port"
)

// ManagerAdapter is an adapter that implements the FileManager port using the standard os package.
type ManagerAdapter struct{}

// Ensure ManagerAdapter implements the FileManager port
var _ port.FileManager = (*ManagerAdapter)(nil)

// NewManagerAdapter creates a new file manager adapter.
func NewManagerAdapter() *ManagerAdapter {
	return &ManagerAdapter{}
}

// ReadFile reads the contents of a file.
func (f *ManagerAdapter) ReadFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return data, nil
}

// WriteFile writes data to a file with specified permissions.
func (f *ManagerAdapter) WriteFile(filename string, data []byte, perm int) error {
	if err := os.WriteFile(filename, data, os.FileMode(perm)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}
	return nil
}

// FileExists checks if a file exists.
func (f *ManagerAdapter) FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
