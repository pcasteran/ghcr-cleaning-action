package main

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-github/v49/github"
	"github.com/rs/zerolog/log"
	"regexp"
)

func clean(ghClient *GithubClient, regClient *ContainerRegistryClient, user, pkg, registry string, prTagRegex *regexp.Regexp, dryRun bool) error {
	// List all the versions of the package.
	log.Debug().Str("user", user).Str("package", pkg).Msg("listing all the package versions")
	pkgVersions, err := ghClient.GetAllContainerPackageVersions(user, pkg)
	if err != nil {
		return fmt.Errorf("unable to list the package versions: %w", err)
	}

	packageVersionByHash := make(map[string]*github.PackageVersion)
	for _, pkgVersion := range pkgVersions {
		packageVersionByHash[*pkgVersion.Name] = pkgVersion
	}

	// Get the registry object (image or image index) for each hash.
	repository := fmt.Sprintf("%s/%s/%s", registry, user, pkg)
	log.Debug().Str("repository", repository).Msg("fetching the container registry objects")
	imageByHash := make(map[string]v1.Image)
	indexByHash := make(map[string]v1.ImageIndex)
	for hash := range packageVersionByHash {
		log.Trace().Str("hash", hash).Msg("fetching container registry object")

		// Get the container registry object.
		image, index, err := regClient.GetRegistryObjectFromHash(repository, hash)
		if err != nil {
			log.Warn().Err(err).Msg("unable to retrieve container registry object")
			continue
		}

		if image != nil {
			imageByHash[hash] = image
		} else if index != nil {
			indexByHash[hash] = index
		} else {
			// Something went wrong, we should never be here...
			log.Warn().Err(err).Msg("invalid container registry object, that should not happen")
			continue
		}
	}

	// Determine the package versions to delete.
	toDelete, err := computeHashesToDelete(ghClient, imageByHash, indexByHash, prTagRegex)
	if err != nil {
		return fmt.Errorf("unable to compute the hashed to delete: %w", err)
	}

	// Delete them.
	if !dryRun {
		// No dry run, perform the deletion.
		nbDeleted := 0
		for _, hash := range toDelete {
			version := packageVersionByHash[hash]
			log.Trace().Str("hash", hash).Int64("version-id", *version.ID).Msg("Deleting package version")
			nbDeleted++ // TODO: increment if no error during deletion
		}

		log.Info().Int("nb-deleted", nbDeleted).Msg("Registry cleaning done")
	} else {
		// Dry run mode, don't perform the deletion.
		log.Info().Msg("Dry run mode is ON, no deletion has been performed")
	}

	return nil
}

func computeHashesToDelete(
	ghClient *GithubClient,
	imageByHash map[string]v1.Image,
	indexByHash map[string]v1.ImageIndex,
	prTagRegex *regexp.Regexp) ([]string, error) {
	// Create a string set containing the hashes to delete.
	toDelete := make(map[string]struct{})

	// Return a slice of the hashes to delete.
	var ret []string
	for hash := range toDelete {
		ret = append(ret, hash)
	}
	return ret, nil
}
