package main

import (
	"log"
	"os"

	"github.com/ionrock/hugs/web"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "hugs",
		Usage: "Hugo blog editor server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   "8080",
				Usage:   "Port to run the server on",
			},
			&cli.StringFlag{
				Name:    "content-dir",
				Aliases: []string{"d"},
				Usage:   "Path to the content/post directory (defaults to ./content/post)",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(c *cli.Context) error {
	// Create a new server
	server, err := web.New(c.String("content-dir"), c.String("port"))
	if err != nil {
		return err
	}

	// Start the server
	return server.Start()
}
