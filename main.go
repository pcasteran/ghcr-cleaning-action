package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/google/go-github/v49/github"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
)

type GithubContainerRegistryRepositoryEntry struct {
	pkgVersion *github.PackageVersion
	manifest   *struct{} // TODO
}

func main() {
	// Parse the command line arguments.
	debug := flag.Bool("debug", false, "Enable the debug logs")
	registry := github.String("ghcr.io")
	user := github.String("pcasteran")
	//password := github.String("")
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
		return
	}

	// List all the versions of the package.
	pkgVersions, err := client.GetAllContainerPackageVersions(*user, *pkg)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("user", *user).
			Str("package", *pkg).
			Msg("unable to list the package versions")
		return
	}

	// Build the Docker registry authentication data.
	auth := &authn.Basic{
		Username: *user,
		Password: *password,
	}

	// Get the manifest for each digest.
	repository := fmt.Sprintf("%s/%s/%s", *registry, *user, *pkg)
	repositoryEntriesByDigest := make(map[string]*GithubContainerRegistryRepositoryEntry)
	for _, pkgVersion := range pkgVersions {
		// Build the digest from the repository and hash.
		hash := *pkgVersion.Name
		fullName := fmt.Sprintf("%s@%s", repository, hash)
		digest, err := name.NewDigest(fullName, name.StrictValidation)
		if err != nil {
			log.Warn().
				Err(err).
				Str("hash", hash).
				Msg("unable to build digest from SHA")
			continue
		}

		// Retrieve the descriptor for the digest.
		descriptor, err := remote.Get(digest, remote.WithAuth(auth))
		if err != nil {
			log.Warn().
				Err(err).
				Stringer("digest", digest).
				Msg("unable to retrieve descriptor")
			continue
		}

		// Check the entry media type.
		mediaType := descriptor.Descriptor.MediaType
		if mediaType != types.DockerManifestSchema2 && mediaType != types.DockerManifestList {
			log.Warn().
				Err(err).
				Stringer("digest", digest).
				Str("media-type", fmt.Sprintf("%v", mediaType)).
				Msg("invalid media type")
			continue
		}

		// Parse the manifest.

		// Add the repository entry.
		repositoryEntriesByDigest[hash] = &GithubContainerRegistryRepositoryEntry{
			pkgVersion: pkgVersion,
		}

		log.Debug().
			Stringer("digest", digest).
			Str("media-type", fmt.Sprintf("%v", mediaType)).
			Msg("registry entry fetched")
	}

	//
	// DockerManifestSchema2: application/vnd.docker.distribution.manifest.v2+json
	// DockerManifestList: "application/vnd.docker.distribution.manifest.list.v2+json"
}
