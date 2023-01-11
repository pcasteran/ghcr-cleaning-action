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
	registry := flag.String("registry", "ghcr.io", "The URL of the container registry")
	user := flag.String("user", "", "The container registry user")
	password := flag.String("password", "", "The container registry user password or access token")
	pkg := flag.String("package", "", "The name of the package to clean")
	prTagRegex := flag.String("pr-tag-regex", "pr-(\\d+).*", "The regex used to match the pull request tags")
	flag.Parse()

	// TODO: temp for test
	b, err := os.ReadFile("pat.txt")
	if err != nil {
		fmt.Print(err)
		return
	}
	password = github.String(string(b))
	_ = prTagRegex

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
