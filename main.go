package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

// moveToTrash moves a file or symlink to the Trash directory.
func moveToTrash(filePath string, verbose bool) error {
	trashFilesPath := getTrashFilesPath()
	trashInfoPath := getTrashInfoPath()
	if verbose {
		println("Found Trash files path:", trashFilesPath)
		println("Found Trash info path:", trashInfoPath)
	}
	destDir := trashFilesPath

	// Create the Trash directories if they don't exist
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(trashInfoPath, os.ModePerm); err != nil {
		return err
	}

	// Get the destination path in the Trash directory
	_, fileName := filepath.Split(filePath)
	destPath := filepath.Join(destDir, fileName)

	// Determine if the file is on a different filesystem
	trashOnDiffFileSystem := false
	if strings.HasPrefix(destDir, "/mnt") {
		trashOnDiffFileSystem = true
	}

	// If the symlink points to a different filesystem, move it safely to Trash
	if trashOnDiffFileSystem {
		err := copyFile(filePath, destPath)
		if err != nil {
			return err
		}
		err = createTrashMetadata(filePath, trashInfoPath, verbose)
		if err != nil {
			return err
		}
		return os.Remove(filePath)
	}
	err := os.Rename(filePath, destPath)
	if err != nil {
		return err
	}
	err = createTrashMetadata(filePath, trashInfoPath, verbose)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <file-path> <verbose> (optional)")
		return
	}

	filePath := os.Args[1]
	verbose := false
	if len(os.Args) == 3 {
		verbose = true
	}

	// Check if file exists
	if _, err := os.Lstat(filePath); os.IsNotExist(err) {
		fmt.Println("File does not exist:", filePath)
		return
	}

	if err := moveToTrash(filePath, verbose); err != nil {
		fmt.Println("Error moving file to Trash:", err)
		return
	}

	if verbose {
		fmt.Println("File moved to Trash successfully.")
	}
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
