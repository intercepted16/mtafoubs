package main

import (
	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
	"os"
)

func initApp() {
	// Initialize the application
	app := &cli.App{
		Name:           "mtafoubs",
		Usage:          "Move to and from the Trash",
		DefaultCommand: "mv",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:     "verbose",
				Usage:    "Print verbose output",
				Required: false,
				Value:    false,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "move",
				Aliases: []string{"mv"},
				Usage:   "Move a file to the Trash",
				Action: func(context *cli.Context) error {
					err := moveToTrash(context)
					if err != nil {
						println(err.Error())
						return err
					}
					return nil
				},
			},
			{
				Name:    "restore",
				Aliases: []string{"res"},
				Usage:   "Restores a file from the Trash",
				Action: func(context *cli.Context) error {
					err := restoreFile(context)
					if err != nil {
						_, err = color.New(color.FgRed).Println(err.Error())
						if err != nil {
							// fallback to println
							println(err.Error())
						}
					}
					return nil

				},
			},
			{
				Name:    "empty",
				Aliases: []string{"emp"},
				Usage:   "Empty the Trash",
				Action: func(context *cli.Context) error {
					err := emptyTrash(context)
					if err != nil {
						_, err = color.New(color.FgRed).Println(err.Error())
						if err != nil {
							// fallback to println
							println(err.Error())
						}
					}
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List files in the Trash",
				Action:  restoreFile,
			},
		},
	}

	// Run the application
	err := app.Run(os.Args)
	if err != nil {
		return
	}
}

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
