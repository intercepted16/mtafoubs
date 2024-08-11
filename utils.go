package main

import (
	"os"
	"path/filepath"
)

func removeContents(dir string) error {
	// Ensure the root directory itself is excluded
	if dir == "" {
		return nil
	}

	// Use WalkDir for Go 1.16 and later
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == dir {
			return nil
		}

		// Check if the entry is a directory or a file
		if d.IsDir() {
			// Remove the directory and its contents
			return os.RemoveAll(path)
		}
		// Remove the file
		return os.Remove(path)
	})

	return err
}
