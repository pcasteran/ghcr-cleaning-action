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

type ContainerRegistryObject struct {
	image v1.Image
	index v1.ImageIndex
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
func (c *ContainerRegistryClient) GetRegistryObjectFromHash(repository, hash string) (*ContainerRegistryObject, error) {
	// Build the digest from the repository and hash.
	objectFullName := fmt.Sprintf("%s@%s", repository, hash)
	digest, err := name.NewDigest(objectFullName, name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("unable to build digest from hash '%s': %w", hash, err)
	}

	// Retrieve the descriptor for the digest.
	descriptor, err := remote.Get(digest, remote.WithAuth(c.auth))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve descriptor from digest '%s': %w", digest, err)
	}

	// Analyse the manifest.
	var image v1.Image
	var index v1.ImageIndex
	switch descriptor.Descriptor.MediaType {
	// Image manifest.
	case types.DockerManifestSchema2:
		image, err = descriptor.Image()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve image from descriptor '%v': %w", descriptor, err)
		}
		break

	// Image index manifest.
	case types.DockerManifestList:
		index, err = descriptor.ImageIndex()
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve image index from descriptor '%v': %w", descriptor, err)
		}
		break

	// Unmanaged manifest type.
	default:
		return nil, fmt.Errorf("unmanaged media type for descriptor '%v': %s", descriptor, descriptor.Descriptor.MediaType)
	}

	return &ContainerRegistryObject{
		image: image,
		index: index,
	}, nil
}
