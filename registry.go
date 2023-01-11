package main

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type ContainerRegistryClient struct {
	auth authn.Authenticator
}

// NewContainerRegistryClient returns an initialized OCI container registry client
func NewContainerRegistryClient(userName, password string) (*ContainerRegistryClient, error) {
	// Build the Docker registry authentication data.
	auth := &authn.Basic{
		Username: userName,
		Password: password,
	}

	return &ContainerRegistryClient{
		auth: auth,
	}, nil
}

// GetRegistryObjectFromHash returns a repository object (image or image index) from its hash.
func (c *ContainerRegistryClient) GetRegistryObjectFromHash(repository, hash string) (v1.Image, v1.ImageIndex, error) {
	// Build the digest from the repository and hash.
	objectFullName := fmt.Sprintf("%s@%s", repository, hash)
	digest, err := name.NewDigest(objectFullName, name.StrictValidation)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to build digest from hash '%s': %w", hash, err)
	}

	// Retrieve the descriptor for the digest.
	descriptor, err := remote.Get(digest, remote.WithAuth(c.auth))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to retrieve descriptor from digest '%s': %w", digest, err)
	}

	// Analyse the manifest.
	switch descriptor.Descriptor.MediaType {
	case types.DockerManifestSchema2:
		// Image manifest.
		image, err := descriptor.Image()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to retrieve image from descriptor '%v': %w", descriptor, err)
		}
		return image, nil, nil

	case types.DockerManifestList:
		// Image index manifest.
		index, err := descriptor.ImageIndex()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to retrieve image index from descriptor '%v': %w", descriptor, err)
		}
		return nil, index, nil

	default:
		// Unmanaged manifest type.
		return nil, nil, fmt.Errorf("unmanaged media type for descriptor '%v': %s", descriptor, descriptor.Descriptor.MediaType)
	}
}
