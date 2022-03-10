package main

import (
	"log"
	"os"

	"github.com/YuuSatoh/implstub"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "implstub",
		Usage: "Selecting interface and struct will result in a temporary implementation.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Value:   "",
				Usage:   "specify the output file path",
			},
			&cli.BoolFlag{
				Name:    "overwrite",
				Aliases: []string{"w"},
				Usage:   "overwrite the specified receiver file",
			},
			&cli.BoolFlag{
				Name:    "pointer",
				Aliases: []string{"p"},
				Usage:   "create a stub with the pointer receiver",
			},
		},
		Action: func(c *cli.Context) error {
			var f *string
			argF := c.String("file")
			if argF != "" {
				f = &argF
			}

			return implstub.Exec(f, c.Bool("overwrite"), c.Bool("pointer"))
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
