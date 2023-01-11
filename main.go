package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/v49/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func main() {
	// Parse the command line arguments.
	debug := flag.Bool("debug", false, "Enable the debug logs")
	registry := github.String("ghcr.io")
	user := github.String("pcasteran")
	//password := github.String("")
	// TODO: PR tag regex
	pkg := github.String("terraform-graph-beautifier")
	flag.Parse()

	// TODO: temp for test
	b, err := os.ReadFile("pat.txt")
	if err != nil {
		fmt.Print(err)
		return
	}
	password := github.String(string(b))

	// Configure the logging.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Create the GitHub client.
	ghClient, err := NewGithubClient(context.Background(), *password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the GitHub client")
	}

	// Create the container registry client.
	regClient, err := NewContainerRegistryClient(*user, *password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the container registry client")
	}

	// Perform the registry cleaning.
	err = clean(ghClient, regClient, *user, *pkg, *registry)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to perform the registry cleaning")
	}
}
