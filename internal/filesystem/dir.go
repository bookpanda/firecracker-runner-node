package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CleanFilesInDir(dir string, filePrefix string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	for _, file := range files {
		if strings.HasPrefix(file.Name(), filePrefix) {
			err = os.RemoveAll(filepath.Join(dir, file.Name()))
			if err != nil {
				return fmt.Errorf("failed to remove file: %v", err)
			}
		}
	}

	return nil
}
