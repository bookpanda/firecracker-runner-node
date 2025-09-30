package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetEmptyLogDir(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Empty the directory if it already existed
	d, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("failed to read log directory: %v", err)
	}

	for _, entry := range d {
		err = os.RemoveAll(filepath.Join(logDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to remove %s: %v", entry.Name(), err)
		}
	}

	return nil
}
