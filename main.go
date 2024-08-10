package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func splitBeforeLast(s string, beforeLast string) (string, string) {
	lastIndex := strings.LastIndex(s, beforeLast)
	if lastIndex == -1 {
		return s, "" // Not found
	}

	secondLastIndex := strings.LastIndex(s[:lastIndex], beforeLast)
	if secondLastIndex == -1 {
		return "", s // Only one found
	}

	return s[:secondLastIndex], s[:lastIndex]
}

// getTrashPath returns the path to the Trash directory.
func getTrashPath() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "Trash", "files")
}

// getTrashInfoPath returns the path to the Trash info directory.
func getTrashInfoPath() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "Trash", "info")
}

// getFileSystemDeviceID returns the device ID of the filesystem containing the path.
func getFileSystemDeviceID(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bfree, nil
}

// copySymlink copies a symlink to the Trash directory.
func copySymlink(srcPath, destPath string) error {
	target, err := os.Readlink(srcPath)
	if err != nil {
		return err
	}
	return os.Symlink(target, destPath)
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

// moveToTrash moves a file or symlink to the Trash directory.
func moveToTrash(filePath string) error {
	trashPath := getTrashPath()
	trashInfoPath := getTrashInfoPath()

	// Create the Trash directories if they don't exist
	if err := os.MkdirAll(trashPath, os.ModePerm); err != nil {
		return err
	}
	if err := os.MkdirAll(trashInfoPath, os.ModePerm); err != nil {
		return err
	}

	// Get the destination path in the Trash directory
	_, fileName := filepath.Split(filePath)
	destPath := filepath.Join(trashPath, fileName)

	// Get filesystem device IDs
	origDev, err := getFileSystemDeviceID(filePath)
	if err != nil {
		return err
	}
	homeDir := os.Getenv("HOME")
	homeDev, err := getFileSystemDeviceID(homeDir)
	if err != nil {
		return err
	}

	// Determine if it's a symlink
	isSymlink := false
	println("Dest path:", destPath)
	destDir, _ := splitBeforeLast(destPath, "/")
	fi, err := os.Lstat(destDir)
	if err != nil {
		return err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		isSymlink = true
	}

	symLinkOnDiffFileSystem := false
	if isSymlink {
		println("Dest pat1h:", destDir)
		destPath, err = os.Readlink(destDir)
		destPath = filepath.Join(destPath, "files", fileName)
		if err != nil {
			return err
		}
		if strings.HasPrefix(destPath, "/mnt") {
			symLinkOnDiffFileSystem = true
		}
	}

	// Handle symlinks
	if isSymlink {
		println("Symlink found")
		println("Dest path:", destPath)
		// If the symlink points to a different filesystem, copy it to Trash
		// Check if the symlink is on a different filesystem
		if symLinkOnDiffFileSystem {
			fmt.Println("Symlink is on a different filesystem, copying to Trash.")
			println("Dest path88:", destDir)

			err := copyFile(filePath, destPath)
			if err != nil {
				return err
			}
			err = createTrashMetadata(filePath, trashInfoPath)
			if err != nil {
				return err
			}
			return os.Remove(filePath)
		}
		print("Symlink is on the same filesystem, moving to Trash.")
	}

	// Handle regular files
	if origDev != homeDev {
		fmt.Println("File is on a different filesystem, copying to Trash.")
		if err := copyFile(filePath, destPath); err != nil {
			return err
		}
		return os.Remove(filePath)
	}
	return os.Rename(filePath, destPath)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <file-path>")
		return
	}

	filePath := os.Args[1]

	// Check if file exists
	if _, err := os.Lstat(filePath); os.IsNotExist(err) {
		fmt.Println("File does not exist:", filePath)
		return
	}

	if err := moveToTrash(filePath); err != nil {
		fmt.Println("Error moving file to Trash:", err)
		return
	}

	fmt.Println("File moved to Trash successfully.")
}

func createTrashMetadata(filePath, trashInfoPath string) error {
	fmt.Println("Creating metadata for file:", filePath, "in Trash info path:", trashInfoPath)
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
