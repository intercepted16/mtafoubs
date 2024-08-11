package main

import (
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	// Initialize the application
	app := &cli.App{
		Name:  "mtafoubs",
		Usage: "Move to and from the Trash",
		Action: func(context *cli.Context) error {
			return errHandler(moveToTrash, context)
		},
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
				Name:    "put",
				Aliases: []string{"mv", "trash"},
				Usage:   "Move a file to the Trash",
				Action: func(context *cli.Context) error {
					return errHandler(moveToTrash, context)
				},
			},
			{
				Name:    "restore",
				Aliases: []string{"res"},
				Usage:   "Restores a file from the Trash",
				Action: func(context *cli.Context) error {
					return errHandler(restoreFile, context)
				},
			},
			{
				Name:    "empty",
				Aliases: []string{"emp"},
				Usage:   "Empty the Trash",
				Action: func(context *cli.Context) error {
					return errHandler(emptyTrash, context)
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List files in the Trash",
				Action: func(context *cli.Context) error {
					return errHandler(listTrash, context)
				},
			},
		},
	}

	// Run the application
	err := app.Run(os.Args)
	if err != nil {
		return
	}
}
