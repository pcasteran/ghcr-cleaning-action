package main

import (
	"context"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"regexp"
	"strings"
)

const defaultPrTagPattern = "^pr-(\\d+).*"

func main() {
	// Parse the command line arguments.
	debug := flag.Bool("debug", false, "Enable the debug logs")
	dryRun := flag.Bool("dry-run", false, "If true, compute everything but do no perform the deletion")
	registry := flag.String("registry", "ghcr.io", "The URL of the container registry")
	user := flag.String("user", "", "The container registry user")
	password := flag.String("password", "", "The container registry user password or access token")
	pkg := flag.String("package", "", "The name of the package to clean")
	repository := flag.String("repository", "", "The GitHub repository (format owner/repository) in which to check the pull requests statuses")
	prTagPattern := flag.String("pr-tag-regex", defaultPrTagPattern, "The regex used to match the pull request tags")
	flag.Parse()

	// Configure the logging.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
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
	ownerAndRepo := strings.Split(*repository, "/")
	pkgRegistryParams := PackageRegistryParams{
		registry: *registry,
		user:     *user,
		pkg:      *pkg,
	}
	prFilterParams := PullRequestFilterParams{
		owner:      ownerAndRepo[0],
		repository: ownerAndRepo[1],
		tagRegex:   regexp.MustCompile(*prTagPattern),
	}
	err = clean(ghClient, prFilterParams, regClient, pkgRegistryParams, *dryRun)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to perform the registry cleaning")
	}
}
