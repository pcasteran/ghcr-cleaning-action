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
	toDelete, err := computeHashesToDelete(ghClient, packageVersionByHash, imageByHash, indexByHash, prTagRegex)
	if err != nil {
		return fmt.Errorf("unable to compute the hashes to delete: %w", err)
	}

	// Delete them.
	if !dryRun {
		// No dry run, perform the deletion.
		nbDeleted := 0
		for _, hash := range toDelete {
			version := packageVersionByHash[hash]
			log.Trace().Str("hash", hash).Int64("version-id", *version.ID).Msg("deleting package version")
			nbDeleted++ // TODO: increment if no error during deletion
		}

		log.Info().Int("nb-deleted", nbDeleted).Msg("registry cleaning done")
	} else {
		// Dry run mode, don't perform the deletion.
		log.Info().Msg("dry run mode is ON, no deletion has been performed")
	}

	return nil
}

func computeHashesToDelete(
	ghClient *GithubClient,
	packageVersionByHash map[string]*github.PackageVersion,
	imageByHash map[string]v1.Image,
	indexByHash map[string]v1.ImageIndex,
	prTagRegex *regexp.Regexp) ([]string, error) {
	// Create the sets containing the hashes to process and to delete.
	toDelete := make(map[string]struct{})

	toProcess := make(map[string]struct{})
	for hash := range packageVersionByHash {
		toProcess[hash] = struct{}{}
	}

	// First, analyse the image indices.
	nbUntaggedIndices := 0
	nbRelatedToClosedPRIndices := 0
	nbReferencedImages := 0
	for hash, index := range indexByHash {
		tags := packageVersionByHash[hash].Metadata.Container.Tags
		deleteIndex := false

		// First, check if the image index is untagged.
		if len(tags) == 0 {
			nbUntaggedIndices++
			deleteIndex = true
		} else {
			// There are tags, check if they are related to a closed pull request.
			isRelatedToClosedPR, err := checkTagsRelatedToClosedPullRequest(ghClient, prTagRegex, tags)
			if err != nil {
				log.Warn().Err(err).Msg("unable to check if image index is related to a closed PR")
			} else if isRelatedToClosedPR {
				nbRelatedToClosedPRIndices++
				deleteIndex = true
			}
		}

		// The index can reference other indices and images:
		//   - the referenced images will also be marked as processed
		//   - the references indices are ignored as they will be processed by the main loop
		indexManifest, err := index.IndexManifest()
		if err != nil {
			log.Warn().Err(err).Msg("unable to get the image index manifest")
			continue
		}

		for _, manifest := range indexManifest.Manifests {
			referencedHash := manifest.Digest.String()
			if _, ok := imageByHash[referencedHash]; ok {
				// Mark the referenced image hash as processed.
				delete(toProcess, referencedHash)

				// Mark the referenced image hash to be deleted.
				if deleteIndex {
					nbReferencedImages++
					toDelete[referencedHash] = struct{}{}
				}
			}
		}

		// Mark the image index hash as processed.
		delete(toProcess, hash)

		// Mark the image index hash to be deleted.
		if deleteIndex {
			toDelete[hash] = struct{}{}
		}
	}

	// Then, analyse the remaining images.
	nbUntaggedImages := 0
	nbRelatedToClosedPRImages := 0
	for hash := range toProcess {
		tags := packageVersionByHash[hash].Metadata.Container.Tags
		deleteImage := false

		// First, check if the image is untagged.
		if len(tags) == 0 {
			nbUntaggedImages++
			deleteImage = true
		} else {
			// There are tags, check if they are related to a closed pull request.
			isRelatedToClosedPR, err := checkTagsRelatedToClosedPullRequest(ghClient, prTagRegex, tags)
			if err != nil {
				log.Warn().Err(err).Msg("unable to check if image is related to a closed PR")
			} else if isRelatedToClosedPR {
				nbRelatedToClosedPRImages++
				deleteImage = true
			}
		}

		// Mark the image hash as processed.
		delete(toProcess, hash)

		// Mark the image hash to be deleted.
		if deleteImage {
			toDelete[hash] = struct{}{}
		}
	}

	// Check that the set of hashes to process is empty.
	if len(toProcess) > 0 {
		var remaining []string
		for hash := range toProcess {
			remaining = append(remaining, hash)
		}
		return nil, fmt.Errorf("some hashes were not processed, that should not happen : %v", remaining)
	}

	// Log the summary.
	log.Debug().
		Int("nb-indices-untagged", nbUntaggedIndices).
		Int("nb-indices-related-to-closed-pr", nbRelatedToClosedPRIndices).
		Int("nb-images-referenced-by-indices", nbReferencedImages).
		Int("nb-images-untagged", nbUntaggedImages).
		Int("nb-images-related-to-closed-pr", nbRelatedToClosedPRImages).
		Msg("hashes to be deleted computed")

	// Return a slice of the hashes to delete.
	var ret []string
	for hash := range toDelete {
		ret = append(ret, hash)
	}
	return ret, nil
}

func checkTagsRelatedToClosedPullRequest(ghClient *GithubClient, prTagRegex *regexp.Regexp, tags []string) (bool, error) {
	return false, nil
}
