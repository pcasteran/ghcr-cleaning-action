package main

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"regexp"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "ghcr-cleaning-action",
	Short: "GitHub action allowing to clean a GitHub Container registry",
	Run:   doExecute,
}

var (
	debug        bool
	dryRun       bool
	registry     string
	user         string
	password     string
	pkg          string
	repository   string
	prTagPattern string
)

const defaultPrTagPattern = "^pr-(\\d+).*"

func init() {
	rootCmd.Flags().BoolVar(&debug, "debug", false, "enable the debug logs")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "if true, compute everything but do no perform the deletion")
	rootCmd.Flags().StringVar(&registry, "registry", "ghcr.io", "the URL of the container registry")
	rootCmd.Flags().StringVar(&user, "user", "", "the container registry user")
	rootCmd.Flags().StringVar(&password, "password", "", "the container registry user password or access token")
	rootCmd.Flags().StringVar(&pkg, "package", "", "the name of the package to clean")
	rootCmd.Flags().StringVar(&repository, "repository", "", "the GitHub repository (format owner/repository) in which to check the pull requests statuses")
	rootCmd.Flags().StringVar(&prTagPattern, "pr-tag-regex", defaultPrTagPattern, "the regular expression used to match the pull request tags, must include one capture group for the PR id")

	_ = rootCmd.MarkFlagRequired("user")
	_ = rootCmd.MarkFlagRequired("password")
	_ = rootCmd.MarkFlagRequired("package")
	_ = rootCmd.MarkFlagRequired("repository")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func doExecute(cmd *cobra.Command, args []string) {
	// Remove unused parameters warning.
	_ = cmd
	_ = args

	// Configure the logging.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Create the GitHub client.
	ghClient, err := NewGithubClient(context.Background(), password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the GitHub client")
	}

	// Create the container registry client.
	regClient, err := NewContainerRegistryClient(user, password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the container registry client")
	}

	// Perform the registry cleaning.
	ownerAndRepo := strings.Split(repository, "/")
	if len(ownerAndRepo) != 2 {
		log.Fatal().Err(err).Msg("invalid repository format, must be owner/repository")
	}

	pkgRegistryParams := PackageRegistryParams{
		registry: registry,
		user:     user,
		pkg:      pkg,
	}
	prFilterParams := PullRequestFilterParams{
		owner:      ownerAndRepo[0],
		repository: ownerAndRepo[1],
		tagRegex:   regexp.MustCompile(prTagPattern),
	}
	err = clean(ghClient, prFilterParams, regClient, pkgRegistryParams, dryRun)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to perform the registry cleaning")
	}
}
