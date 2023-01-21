package cmd

import (
	"context"
	"fmt"
	"github.com/pcasteran/ghcr-cleaning-action/pkg"
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
	packageName  string
	repository   string
	prTagPattern string
)

func init() {
	rootCmd.Flags().BoolVar(&debug, "debug", false, "enable the debug logs")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "if true, compute everything but do no perform the deletion")
	rootCmd.Flags().StringVar(&registry, "registry", "ghcr.io", "the URL of the container registry")
	rootCmd.Flags().StringVar(&user, "user", "", "the container registry user")
	rootCmd.Flags().StringVar(&password, "password", "", "the container registry user password or access token")
	rootCmd.Flags().StringVar(&packageName, "package", "", "the name of the package to clean")
	rootCmd.Flags().StringVar(&repository, "repository", "", "the GitHub repository (format owner/repository) in which to check the pull requests statuses")
	rootCmd.Flags().StringVar(&prTagPattern, "pr-tag-regex", pkg.DefaultPrTagPattern, "the regular expression used to match the pull request tags, must include one capture group for the PR id")

	_ = rootCmd.MarkFlagRequired("user")
	_ = rootCmd.MarkFlagRequired("password")
	_ = rootCmd.MarkFlagRequired("package")
	_ = rootCmd.MarkFlagRequired("repository")
}

func Execute() {
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
	ghClient, err := pkg.NewGithubClient(context.Background(), password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the GitHub client")
	}

	// Create the container registry client.
	regClient, err := pkg.NewContainerRegistryClient(user, password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the container registry client")
	}

	// Perform the registry cleaning.
	ownerAndRepo := strings.Split(repository, "/")
	if len(ownerAndRepo) != 2 {
		log.Fatal().Err(err).Msg("invalid repository format, must be owner/repository")
	}

	pkgRegistryParams := pkg.PackageRegistryParams{
		Registry:    registry,
		User:        user,
		PackageName: packageName,
	}
	prFilterParams := pkg.PullRequestFilterParams{
		Owner:      ownerAndRepo[0],
		Repository: ownerAndRepo[1],
		TagRegex:   regexp.MustCompile(prTagPattern),
	}
	err = pkg.Clean(ghClient, prFilterParams, regClient, pkgRegistryParams, dryRun)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to perform the registry cleaning")
	}
}
