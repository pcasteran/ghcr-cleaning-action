package main

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-github/v49/github"
	"github.com/rs/zerolog/log"
	"regexp"
)

func clean(ghClient *GithubClient, regClient *ContainerRegistryClient, user, pkg, registry string, prTagRegex *regexp.Regexp) error {
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
	log.Debug().Str("user", user).Str("package", pkg).Msg("fetching the container registry objects")
	repository := fmt.Sprintf("%s/%s/%s", registry, user, pkg)
	imageByHash := make(map[string]v1.Image)
	imageIndexByHash := make(map[string]v1.ImageIndex)
	for hash := range packageVersionByHash {
		log.Trace().Str("hash", hash).Msg("fetching container registry object")

		// Get the container registry object.
		image, index, err := regClient.GetRegistryObjectFromHash(repository, hash)
		if err != nil {
			log.Warn().Err(err).Msg("unable to retrieve container object")
			continue
		}

		if image != nil {
			imageByHash[hash] = image
		}

		if index != nil {
			imageIndexByHash[hash] = index
		}
	}

	return nil
}
