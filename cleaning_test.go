package main

import (
	"errors"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/fake"
	"github.com/google/go-github/v49/github"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"regexp"
	"testing"
)

//
// Test suite definition.
//

type CleaningTestSuite struct {
	suite.Suite
}

func TestCleaningTestSuite(t *testing.T) {
	suite.Run(t, new(CleaningTestSuite))
}

//
// Constants used in the tests.
//

const (
	// Image hash constants.

	image1 = "sha256:3d65e9efc7caafb46aa581c1e00ea8d423c081d31cd59af3bb07bd1d6aa5cd37"
	image2 = "sha256:67fd0c23255eaf9e1cc33aca558ec95c187f30af566a726e23e321b63067b5b8"

	// Image index hash constants.
	index1 = "sha256:50f220674b599fbe570300bae678f2d36eda173eb06115f072a334d6731b30f1"
	index2 = "sha256:627e7a284dd04d9532bab7897077668416c4912d85a08cb7988f8bc547fbc013"
)

var defaultPrFilterParams = PullRequestFilterParams{
	tagRegex: regexp.MustCompile(defaultPrTagPattern),
}

//
// GithubClient mock.
//

type githubClientMock struct {
	mock.Mock
}

func (m *githubClientMock) GetAllContainerPackages(user string) ([]*github.Package, error) {
	_ = user
	return nil, nil
}

func (m *githubClientMock) GetAllContainerPackageVersions(user, packageName string) ([]*github.PackageVersion, error) {
	_ = user
	_ = packageName
	return nil, nil
}

func (m *githubClientMock) GetPullRequestState(owner, repository string, id int) (string, error) {
	// Records that the method was called with its parameters.
	args := m.Called(owner, repository, id)

	// Return whatever we must return.
	return args.String(0), args.Error(1)
}

//
// Tests.
//

func (s *CleaningTestSuite) TestImageNoTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: nil, references: nil},
	})

	toDelete, err := computeHashesToDelete(nil, PullRequestFilterParams{}, versions, images, indices)

	// Check the result
	r := s.Require()
	r.NoError(err)
	r.ElementsMatch(toDelete, []string{image1})
}

func (s *CleaningTestSuite) TestImageValidTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: []string{"v1.2.3"}, references: nil},
	})

	toDelete, err := computeHashesToDelete(nil, defaultPrFilterParams, versions, images, indices)

	// Check the result
	r := s.Require()
	r.NoError(err)
	r.Empty(toDelete)
}

func (s *CleaningTestSuite) TestImageActivePullRequestTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: []string{"pr-1234"}, references: nil},
	})

	ghClient := new(githubClientMock)
	ghClient.
		On("GetPullRequestState", defaultPrFilterParams.owner, defaultPrFilterParams.repository, 1234).
		Return("active", nil)

	toDelete, err := computeHashesToDelete(ghClient, defaultPrFilterParams, versions, images, indices)

	// Check the result
	ghClient.AssertExpectations(s.T())

	r := s.Require()
	r.NoError(err)
	r.Empty(toDelete)
}

func (s *CleaningTestSuite) TestImageClosedPullRequestTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: []string{"pr-1234"}, references: nil},
	})

	ghClient := new(githubClientMock)
	ghClient.
		On("GetPullRequestState", defaultPrFilterParams.owner, defaultPrFilterParams.repository, 1234).
		Return("closed", nil)

	toDelete, err := computeHashesToDelete(ghClient, defaultPrFilterParams, versions, images, indices)

	// Check the result
	ghClient.AssertExpectations(s.T())

	r := s.Require()
	r.NoError(err)
	r.ElementsMatch(toDelete, []string{image1})
}

func (s *CleaningTestSuite) TestImageUnknownPullRequestTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: []string{"pr-1234"}, references: nil},
	})

	ghClient := new(githubClientMock)
	ghClient.
		On("GetPullRequestState", defaultPrFilterParams.owner, defaultPrFilterParams.repository, 1234).
		Return("", errors.New("not found"))

	toDelete, err := computeHashesToDelete(ghClient, defaultPrFilterParams, versions, images, indices)

	// Check the result
	ghClient.AssertExpectations(s.T())

	r := s.Require()
	r.NoError(err)
	r.Empty(toDelete)
}

func (s *CleaningTestSuite) TestImageMixedActiveAndClosedPullRequestsTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: []string{"pr-1234", "pr-5678"}, references: nil},
	})

	ghClient := new(githubClientMock)
	ghClient.
		On("GetPullRequestState", defaultPrFilterParams.owner, defaultPrFilterParams.repository, 1234).
		Return("closed", nil).
		On("GetPullRequestState", defaultPrFilterParams.owner, defaultPrFilterParams.repository, 5678).
		Return("active", nil)

	toDelete, err := computeHashesToDelete(ghClient, defaultPrFilterParams, versions, images, indices)

	// Check the result
	ghClient.AssertExpectations(s.T())

	r := s.Require()
	r.NoError(err)
	r.Empty(toDelete)
}

func (s *CleaningTestSuite) TestImageMixedValidTagAndClosedPullRequestsTag() {
	// Compute the hashes to delete.
	versions, images, indices := s.buildTestData(map[string]TestDataItem{
		image1: {tags: []string{"pr-1234", "v1.2.3"}, references: nil},
	})

	ghClient := new(githubClientMock)
	ghClient.
		On("GetPullRequestState", defaultPrFilterParams.owner, defaultPrFilterParams.repository, 1234).
		Return("closed", nil)

	toDelete, err := computeHashesToDelete(ghClient, defaultPrFilterParams, versions, images, indices)

	// Check the result
	ghClient.AssertExpectations(s.T())

	r := s.Require()
	r.NoError(err)
	r.Empty(toDelete)
}

// index (tag) => index (no tag) => images

//
// Test data generation.
//

type TestDataItem struct {
	// The tags associated to the items.
	tags []string

	// If `references` is not empty, item is considered to be an index, otherwise it is an image.
	references []string
}

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
