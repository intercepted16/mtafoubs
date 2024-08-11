package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func restoreFile(c *cli.Context) error {
	fileToRestore, err := filepath.Abs(c.Args().First())
	if err != nil {
		return err
	}

	if fileToRestore == "" {
		return fmt.Errorf("file path is required")
	}
	verbose := c.Bool("verbose")
	if verbose {
		println("Restoring file:", fileToRestore)
	}
	// Get the Trash files and info directories
	trashFilesPath := getTrashFilesPath()
	trashInfoPath := getTrashInfoPath()
	if verbose {
		println("Found Trash files path:", trashFilesPath)
		println("Found Trash info path:", trashInfoPath)
	}
	// Get the Trash info file path
	fileToRestoreName := filepath.Base(fileToRestore)
	trashInfoFilePath := filepath.Join(trashInfoPath, fileToRestoreName+".trashinfo")
	// Check if the Trash info file exists
	if _, err := os.Lstat(trashInfoFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file not found in Trash")
	}
	// Read the Trash info file
	trashInfoFile, err := os.ReadFile(trashInfoFilePath)
	if err != nil {
		return err
	}
	// Parse the Trash info file
	trashInfo := strings.Split(string(trashInfoFile), "\n")
	// Get the original file path
	originalFilePath := strings.Split(trashInfo[1], "=")[1]
	// Get the deletion date
	deletionDate := strings.Split(trashInfo[2], "=")[1]
	if verbose {
		fmt.Println("Original file path:", originalFilePath)
		fmt.Println("Deletion date:", deletionDate)
	}
	// Make sure that the original file path matches the file to restore
	// the path to restore should be the same as the original file path
	_, originalFileName := path.Split(originalFilePath)
	_, restoreFileName := path.Split(fileToRestore)
	if originalFileName != restoreFileName {
		return fmt.Errorf("file path does not match the original")
	}
	// Get the Trash files directory
	trashFilesDir, err := os.ReadDir(trashFilesPath)
	if err != nil {
		return err
	}
	// Restore the file
	for _, entry := range trashFilesDir {
		println("entry.Name():", entry.Name())
		if entry.Name() == fileToRestoreName {
			if verbose {
				fmt.Println("Found file in Trash:", fileToRestore)
			}
			// Get the original file path
			originalFilePath := filepath.Join(trashFilesPath, entry.Name())
			// Get the destination path
			destPath := fileToRestore
			// Move the file back to the original location
			err := copyFile(originalFilePath, destPath)
			if err != nil {
				return err
			}

			// Remove the Trash info file
			err = os.Remove(trashInfoFilePath)
			if err != nil {
				return err
			}
			// Remove the file from the Trash
			err = os.Remove(originalFilePath)
			if err != nil {
				return err
			}
			if verbose {
				fmt.Println("File restored successfully.")
			}
			return nil
		}
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

// moveToTrash moves a file or symlink to the Trash directory.
func moveToTrash(c *cli.Context) error {
	filePath := c.Args().First()
	println("filePath:", filePath)
	if filePath == "" {
		return fmt.Errorf("file path is required")
	}
	verbose := c.Bool("verbose")
	println("verbose:", verbose)
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
	initApp()
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
