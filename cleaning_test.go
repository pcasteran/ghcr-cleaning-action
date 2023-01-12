package main

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-github/v49/github"
	"github.com/stretchr/testify/suite"
	"testing"
)

type CleaningTestSuite struct {
	suite.Suite
}

func TestCleaningTestSuite(t *testing.T) {
	suite.Run(t, new(CleaningTestSuite))
}

type TestDataItem struct {
	// The tags associated to the items.
	tags []string

	// If `references` is not empty, item is considered to be an index, otherwise it is an image.
	references []string
}

const (
	// Image hash constants.

	image1 = "sha256:3d65e9efc7caafb46aa581c1e00ea8d423c081d31cd59af3bb07bd1d6aa5cd37"
	image2 = "sha256:67fd0c23255eaf9e1cc33aca558ec95c187f30af566a726e23e321b63067b5b8"

	// Image index hash constants.
	index1 = "sha256:50f220674b599fbe570300bae678f2d36eda173eb06115f072a334d6731b30f1"
	index2 = "sha256:627e7a284dd04d9532bab7897077668416c4912d85a08cb7988f8bc547fbc013"
)

func (s *CleaningTestSuite) buildTestData(items map[string]TestDataItem) (
	map[string]*github.PackageVersion,
	map[string]v1.Image,
	map[string]v1.ImageIndex,
) {
	// Create the package versions.
	packageVersionByHash := make(map[string]*github.PackageVersion)
	for hash, item := range items {
		packageVersionByHash[hash] = &github.PackageVersion{
			Metadata: &github.PackageMetadata{
				Container: &github.PackageContainerMetadata{
					Tags: item.tags,
				},
			},
		}
	}

	// Create the images.
	imageByHash := make(map[string]v1.Image)
	for hash, item := range items {
		if len(item.references) == 0 {
			imageByHash[hash] = &fake.FakeImage{}
		}
	}

	// Create the image indices.
	indexByHash := make(map[string]v1.ImageIndex)
	for hash, item := range items {
		if len(item.references) > 0 {
			// Create the referenced manifests.
			manifests := make([]v1.Descriptor, len(item.references))
			for i, ref := range item.references {
				hash, _ := v1.NewHash(ref)
				manifests[i] = v1.Descriptor{
					Digest: hash,
				}
			}

			// Create the image index.
			indexByHash[hash] = &fake.FakeImageIndex{
				IndexManifestStub: func() (*v1.IndexManifest, error) {
					return &v1.IndexManifest{
						Manifests: manifests,
					}, nil
				},
			}
		}
	}

	return packageVersionByHash, imageByHash, indexByHash
}

func (s *CleaningTestSuite) TestBuildTestData() {
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: nil, references: nil},
		image2: {tags: []string{"tag1", "tag2"}, references: nil},
		index1: {tags: nil, references: []string{image1, image2}},
		index2: {tags: []string{"tag3"}, references: []string{index1}},
	})

	r := s.Require()

	// Check the package versions.
	r.Len(versions, 4)
	r.Empty(versions[image1].Metadata.Container.Tags)
	r.ElementsMatch(versions[image2].Metadata.Container.Tags, []string{"tag1", "tag2"})
	r.Empty(versions[index1].Metadata.Container.Tags)
	r.ElementsMatch(versions[index2].Metadata.Container.Tags, []string{"tag3"})

	// Check the images.
	r.Len(images, 2)
	r.Contains(images, image1)
	r.Contains(images, image2)

	// Check the image indices.
	r.Len(indices, 2)

	m1, _ := indices[index1].IndexManifest()
	r.Len(m1.Manifests, 2)
	image1Hash, _ := v1.NewHash(image1)
	image2Hash, _ := v1.NewHash(image2)
	r.ElementsMatch(m1.Manifests, []v1.Descriptor{{Digest: image1Hash}, {Digest: image2Hash}})

	m2, _ := indices[index2].IndexManifest()
	r.Len(m2.Manifests, 1)
	index1Hash, _ := v1.NewHash(index1)
	r.ElementsMatch(m2.Manifests, []v1.Descriptor{{Digest: index1Hash}})
}

// Interface githubclient + struct impl
// buildGitHubClientMock(map[string]string prStatuses)

// index (tag) => index (no tag) => images
// tag: something-pr-123
