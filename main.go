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

type GithubContainerRegistryRepositoryEntry struct {
	pkgVersion     *github.PackageVersion
	registryObject *ContainerRegistryObject
}

func main() {
	// Parse the command line arguments.
	debug := flag.Bool("debug", false, "Enable the debug logs")
	registryUrl := github.String("ghcr.io")
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

	// Configure logging.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Create the GitHub client.
	client, err := NewGithubClient(context.Background(), *password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the GitHub client")
	}

	// List all the versions of the package.
	log.Debug().Str("user", *user).Str("package", *pkg).Msg("listing all package versions")
	pkgVersions, err := client.GetAllContainerPackageVersions(*user, *pkg)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to list the package versions")
	}

	// Build the container registry client.
	registryClient, err := NewContainerRegistryClient(*user, *password)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create the container registry client")
	}

	// Get the registry object for each digest.
	repository := fmt.Sprintf("%s/%s/%s", *registryUrl, *user, *pkg)
	repositoryEntriesByDigest := make(map[string]*GithubContainerRegistryRepositoryEntry)
	for _, pkgVersion := range pkgVersions {
		hash := *pkgVersion.Name
		log.Debug().Str("hash", hash).Msg("fetching container registry entry")

		// Get the container registry object.
		object, err := registryClient.GetRegistryObjectFromHash(repository, hash)
		if err != nil {
			log.Warn().Err(err).Msg("unable to retrieve container object")
			continue
		}

		// Add the repository entry.
		repositoryEntriesByDigest[hash] = &GithubContainerRegistryRepositoryEntry{
			pkgVersion:     pkgVersion,
			registryObject: object,
		}
	}
}
