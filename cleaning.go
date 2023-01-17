package main

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-github/v49/github"
	"github.com/rs/zerolog/log"
	"regexp"
	"strconv"
)

type PullRequestFilterParams struct {
	owner      string
	repository string
	tagRegex   *regexp.Regexp
}

type PackageRegistryParams struct {
	registry string
	user     string
	pkg      string
}

func clean(ghClient GithubClient, prFilterParams PullRequestFilterParams, regClient ContainerRegistryClient, pkgRegistryParams PackageRegistryParams, dryRun bool) error {
	// List all the versions of the package.
	log.Debug().Str("user", pkgRegistryParams.user).Str("package", pkgRegistryParams.pkg).Msg("listing all the package versions")
	pkgVersions, err := ghClient.GetAllContainerPackageVersions(pkgRegistryParams.user, pkgRegistryParams.pkg)
	if err != nil {
		return fmt.Errorf("unable to list the package versions: %w", err)
	}

	packageVersionByHash := make(map[string]*github.PackageVersion)
	for _, pkgVersion := range pkgVersions {
		packageVersionByHash[*pkgVersion.Name] = pkgVersion
	}

	// Get the registry object (image or image index) for each hash.
	repository := fmt.Sprintf("%s/%s/%s", pkgRegistryParams.registry, pkgRegistryParams.user, pkgRegistryParams.pkg)
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

	// Determine the hashes to delete.
	toDelete, err := computeHashesToDelete(ghClient, prFilterParams, packageVersionByHash, imageByHash, indexByHash)
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
			err := ghClient.DeleteContainerPackageVersion(pkgRegistryParams.user, pkgRegistryParams.pkg, *version.ID)
			if err != nil {
				log.Warn().Err(err).Msg("unable to delete package version")
				continue
			}
			nbDeleted++
		}

		log.Info().Int("nb-deleted", nbDeleted).Msg("registry cleaning done")
	} else {
		// Dry run mode, don't perform the deletion.
		log.Info().Msg("dry run mode is ON, no deletion has been performed")
	}

	return nil
}

func computeHashesToDelete(
	ghClient GithubClient,
	prFilterParams PullRequestFilterParams,
	packageVersionByHash map[string]*github.PackageVersion,
	imageByHash map[string]v1.Image,
	indexByHash map[string]v1.ImageIndex) ([]string, error) {
	// Create a tree of the registry items.
	type RegistryItem struct {
		referencedCount int
		references      []*RegistryItem
		mustKeep        bool
	}

	items := make(map[string]*RegistryItem)

	// Add the images.
	for hash := range imageByHash {
		items[hash] = &RegistryItem{
			referencedCount: 0,
			references:      nil,
			mustKeep:        hasValidTags(ghClient, prFilterParams, packageVersionByHash[hash].Metadata.Container.Tags),
		}
	}

	// Add the image indices.
	for hash := range indexByHash {
		items[hash] = &RegistryItem{
			referencedCount: 0,
			references:      nil,
			mustKeep:        hasValidTags(ghClient, prFilterParams, packageVersionByHash[hash].Metadata.Container.Tags),
		}
	}

	// Add the references.
	for hash, index := range indexByHash {
		indexManifest, err := index.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("unable to get the image index manifest: %w", err)
		}

		for _, manifest := range indexManifest.Manifests {
			// Get the referenced item.
			referencedHash := manifest.Digest.String()
			referencedItem := items[referencedHash]

			// Add it to the current item references.
			items[hash].references = append(items[hash].references, referencedItem)

			// Increment the references counter on the referenced item.
			referencedItem.referencedCount++
		}
	}

	// Identify the items to be deleted.
	toDelete := make(map[string]struct{})

	nPass := 0
	for {
		nPass++
		nMarkedToDelete := 0

		for hash, item := range items {
			if item.referencedCount == 0 && !item.mustKeep {
				// The current item can be deleted.
				delete(items, hash)
				toDelete[hash] = struct{}{}
				nMarkedToDelete++

				// Decrement the referenced count in all the referenced items.
				for _, ref := range item.references {
					ref.referencedCount--
				}
			}
		}

		log.Debug().Int("pass", nPass).Int("nb-marked-to-delete", nMarkedToDelete).Send()

		if nMarkedToDelete == 0 {
			// Done, there is no more items to mark for deletion.
			break
		}
	}

	// Return a slice of the hashes to delete.
	var ret []string
	for hash := range toDelete {
		ret = append(ret, hash)
	}
	return ret, nil
}

func hasValidTags(ghClient GithubClient, prFilterParams PullRequestFilterParams, tags []string) bool {
	hasValidTags := true

	if len(tags) == 0 {
		hasValidTags = false
	} else {
		// There are tags, check if they are related to a closed pull request.
		isRelatedToClosedPR, err := checkTagsRelatedToClosedPullRequest(ghClient, prFilterParams, tags)
		if err != nil {
			// Error occurred, don't change the returned value as we don't want to delete this object.
			log.Warn().Err(err).Msg("unable to check if a tag is related to a closed PR")
		} else if isRelatedToClosedPR {
			// All the tags are related to a closed pull request.
			hasValidTags = false
		}
	}

	return hasValidTags
}

func checkTagsRelatedToClosedPullRequest(ghClient GithubClient, prFilterParams PullRequestFilterParams, tags []string) (bool, error) {
	// Check if all tags are related to a closed pull request.
	allTagsRelatedToClosedPR := true
	for _, tag := range tags {
		matches := prFilterParams.tagRegex.FindStringSubmatch(tag)
		if matches != nil {
			// Get the pull request id.
			idStr := matches[1]
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return false, fmt.Errorf("unable to parse pull request identifier '%s': %w", idStr, err)
			}

			// Get the pull request status.
			status, err := ghClient.GetPullRequestState(prFilterParams.owner, prFilterParams.repository, id)
			if err != nil {
				return false, fmt.Errorf("unable to retrieve pull request status: %w", err)
			}

			if status != "closed" {
				allTagsRelatedToClosedPR = false
				break
			}
		} else {
			allTagsRelatedToClosedPR = false
			break
		}
	}

	return allTagsRelatedToClosedPR, nil
}
