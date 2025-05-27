package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
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
			&cli.BoolFlag{
				Name:    "hugo-server",
				Aliases: []string{"s"},
				Usage:   "Start the local Hugo server alongside the editor",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err)
	}
}

// startHugoServer starts the local Hugo server in development mode
func startHugoServer(contentDir string) {
	log.Info().Msg("Starting Hugo server")

	// Execute the hugo server command
	cmd := exec.Command("hugo", "server", "-D")

	// Set the command to run in the directory containing the content
	if contentDir != "" {
		// Extract the base directory (removing content/post from the path)
		baseDir := filepath.Dir(filepath.Dir(contentDir))
		log.Debug().Str("hugo_dir", baseDir).Msg("Running Hugo server in directory")
		cmd.Dir = baseDir
	} else {
		// Default to current directory if no content dir specified
		cmd.Dir = "."
	}

	// Redirect stdout and stderr to our logger
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stdout pipe for Hugo server")
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get stderr pipe for Hugo server")
		return
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start Hugo server")
		return
	}

	// Log stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Info().Str("source", "hugo").Msg(scanner.Text())
		}
	}()

	// Log stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Error().Str("source", "hugo").Msg(scanner.Text())
		}
	}()

	// Wait for the command to finish
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Error().Err(err).Msg("Hugo server exited with error")
		} else {
			log.Info().Msg("Hugo server exited")
		}
	}()

	log.Info().Msg("Hugo server started at http://localhost:1313/")
}

func runServer(c *cli.Context) error {
	// Set debug level if requested
	if c.Bool("debug") {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}

	// Start Hugo server if requested
	if c.Bool("hugo-server") {
		go startHugoServer(c.String("content-dir"))
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
