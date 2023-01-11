package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/v49/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	ctx    context.Context
	client *github.Client
}

// NewGithubClient returns an initialized GitHub client
func NewGithubClient(ctx context.Context, token string) (*GithubClient, error) {
	// Create a new http.Client that will manage the authentication.
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, tokenSource)

	// Create the GitHub client.
	githubClient := github.NewClient(httpClient)

	return &GithubClient{
		ctx:    ctx,
		client: githubClient,
	}, nil
}

// GetAllContainerPackages returns all the active packages of type container
func (gh *GithubClient) GetAllContainerPackages(user string) ([]*github.Package, error) {
	// Create an empty list of GitHub packages.
	var packages []*github.Package

	// List all the active packages of type container.
	listOptions := &github.PackageListOptions{
		PackageType: github.String("container"),
		State:       github.String("active"),
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		// Get the next page.
		pkgs, response, err := gh.client.Users.ListPackages(gh.ctx, user, listOptions)
		if err != nil {
			return nil, fmt.Errorf("unable to list container packages for user '%s': %w", user, err)
		}

		// Add the page content to the result list.
		packages = append(packages, pkgs...)

		// Check if there is another page to fetch.
		if response.NextPage == 0 {
			break
		}
		listOptions.Page = response.NextPage
	}

	return packages, nil
}

// GetAllContainerPackageVersions returns all the versions of a package of type container
func (gh *GithubClient) GetAllContainerPackageVersions(user, packageName string) ([]*github.PackageVersion, error) {
	// Create an empty list of GitHub package versions.
	var packageVersions []*github.PackageVersion

	// List all the active packages of type container.
	listOptions := &github.PackageListOptions{
		PackageType: github.String("container"),
		State:       github.String("active"),
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		// Get the next page.
		pkgVersions, response, err := gh.client.Users.PackageGetAllVersions(
			gh.ctx,
			user,
			*listOptions.PackageType,
			packageName,
			listOptions,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to list container package versions for user '%s' and package '%s': %w", user, packageName, err)
		}

		// Add the page content to the result list.
		packageVersions = append(packageVersions, pkgVersions...)

		// Check if there is another page to fetch.
		if response.NextPage == 0 {
			break
		}
		listOptions.Page = response.NextPage
	}

	return packageVersions, nil
}

func (gh *GithubClient) GetPullRequestState(owner, repository string, id int) (string, error) {
	// Get the pull request.
	pr, _, err := gh.client.PullRequests.Get(gh.ctx, owner, repository, id)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve pull request for owner '%s', repository '%s', , id '%d': %w", owner, repository, id, err)
	}

	return *pr.State, nil
}
