package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func parseTrashInfoFile(trashInfoFilePath string) (string, string, error) {
	trashInfoRaw, err := os.ReadFile(trashInfoFilePath)
	if err != nil {
		println("err parsing trashInfoFile:", err.Error())
		return "", "", err
	}

	// Parse the Trash info file
	trashInfo := strings.Split(string(trashInfoRaw), "\n")
	// Get the original file path
	originalFilePath := strings.Split(trashInfo[1], "=")[1]
	// Get the deletion date
	deletionDate := strings.Split(trashInfo[2], "=")[1]
	return originalFilePath, deletionDate, nil
}

func createTrashMetadata(filePath, trashInfoPath string, verbose bool) error {
	if verbose {
		fmt.Println("Creating metadata for file", filePath, "in", trashInfoPath)
	}
	// Extract the file name from the file path
	_, fileName := filepath.Split(filePath)

	// Create the metadata content
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	header := "[Trash Info]\n"
	originalPath := fmt.Sprintf("Path=%s\n", absPath)
	deletionDate := fmt.Sprintf("DeletionDate=%s\n", strings.Split(time.Now().Format(time.RFC3339), "+")[0])

	// Combine the metadata content
	metadataContent := header + originalPath + deletionDate

	// Define the metadata file path
	metadataFilePath := filepath.Join(trashInfoPath, fileName+".trashinfo")

	// Write the metadata to the file
	err = os.WriteFile(metadataFilePath, []byte(metadataContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

// getTrashPath returns the path to the Trash directory.
func getTrashPath() (string, error) {
	trashDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "Trash")
	// Determine if it's a symlink
	isSymlink, err := checkIsSymlink(trashDir)
	if err != nil {
		return "", err
	}

	if isSymlink {
		trashDir, err = os.Readlink(trashDir)
		if err != nil {
			return "", err
		}
	}
	return trashDir, nil
}

// getTrashFilesPath returns the path to the Trash files directory.
func getTrashFilesPath() string {
	trashDir, err := getTrashPath()
	if err != nil {
		panic(err)
	}
	return filepath.Join(trashDir, "files")
}

// getTrashInfoPath returns the path to the Trash info directory.
func getTrashInfoPath() string {
	trashDir, err := getTrashPath()
	if err != nil {
		panic(err)
	}
	return filepath.Join(trashDir, "info")
}

// copyFile copies a file from source to destination.
func copyFile(srcPath, destPath string) error {
	sourceFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func(sourceFile *os.File) {
		err := sourceFile.Close()
		if err != nil {
			panic(err)
		}
	}(sourceFile)

	destinationFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func(destinationFile *os.File) {
		err := destinationFile.Close()
		if err != nil {
			panic(err)
		}
	}(destinationFile)

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

func checkIsSymlink(filePath string) (bool, error) {
	fi, err := os.Lstat(filePath)
	if err != nil {
		return false, err
	}
	return fi.Mode()&os.ModeSymlink != 0, nil
}

func errHandler(f func(ctx *cli.Context) error, ctx *cli.Context) error {
	err := f(ctx)
	if err != nil {
		_, err := color.New(color.FgRed).Println(err.Error())
		if err != nil {
			// fallback to println
			println(err.Error())
		}
	}
	return nil
}
