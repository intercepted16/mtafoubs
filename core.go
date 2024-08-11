// This file contains the logic for:
// moving files to the Trash, restoring files from the Trash,
// emptying the Trash and listing files in the Trash
package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func emptyTrash(context *cli.Context) error {
	// Get the Trash directory
	trashFilesPath := getTrashFilesPath()
	trashInfoPath := getTrashInfoPath()
	verbose := context.Bool("verbose")
	if verbose {
		println("Emptying Trash")
	}
	if verbose {
		println("Found Trash files path:", trashFilesPath)
		println("Found Trash info path:", trashInfoPath)
	}
	// Remove the contents of the Trash directory
	err := removeContents(trashFilesPath)
	if err != nil {
		return err
	}
	// Remove the contents of the Trash info directory
	err = removeContents(trashInfoPath)
	if err != nil {
		return err
	}
	return nil
}

func listTrash(context *cli.Context) error {
	// Get the Trash directory
	trashFilesPath := getTrashFilesPath()
	trashInfoPath := getTrashInfoPath()
	verbose := context.Bool("verbose")
	if verbose {
		println("Listing Trash")
	}
	if verbose {
		println("Found Trash files path:", trashFilesPath)
		println("Found Trash info path:", trashInfoPath)
	}
	// Loop through the Trash directory, find it's corresponding info file and print the details
	err := filepath.Walk(trashFilesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip the root directory itself
		if path == trashFilesPath {
			return nil
		}
		// Get the file name
		_, fileName := filepath.Split(path)
		// Get the info file path
		infoFilePath := filepath.Join(trashInfoPath, fileName+".trashinfo")
		// Check if the info file exists
		_, err = os.Stat(infoFilePath)
		if err != nil {
			return err
		}

		// Parse the info file
		originalPath, delDate, err := parseTrashInfoFile(infoFilePath)
		if err != nil {
			return err
		}
		parsedDelDate, err := time.Parse("2006-01-02T15:04:05", delDate)
		if err != nil {
			// Handle the error
			return err
		}
		// Reassign the deletion date and strip out the +0000 timezone
		delDate = parsedDelDate.Format("2006-01-02 15:04:05")

		println(delDate, originalPath)

		return nil
	},
	)
	if err != nil {
		return err
	}
	return nil
}

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

func moveToTrash(c *cli.Context) error {
	filePath := c.Args().First()
	if filePath == "" {
		return fmt.Errorf("file path is required")
	}
	verbose := c.Bool("verbose")
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
