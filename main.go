package main

import (
	"os"
	"time"

	"github.com/ionrock/hugs/web"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func main() {
	// Configure zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})
	
	// Default log level is info
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
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
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"v"},
				Usage:   "Enable debug logging",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}

func runServer(c *cli.Context) error {
	// Set debug level if requested
	if c.Bool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}
	// Create a new server
	server, err := web.New(c.String("content-dir"), c.String("port"))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create server")
		return err
	}

	// Start the server
	log.Info().Msg("Starting server")
	return server.Start()
}
