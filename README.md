# GitHub Container registry cleaning action

GitHub action allowing to clean a GitHub Container registry by deleting the unnecessary images and
image [indices](https://docs.docker.com/registry/spec/manifest-v2-2/). Unnecessary can mean:

- untagged images not referenced by any image index
- untagged image indices and their referenced images
- tagged images related to a closed Pull Request
- tagged image indices related to a closed Pull Request and their referenced images

There are actually many possible combinations, for a list of all the managed cases see
the [unit tests](cleaning_test.go).

## Basic usage

```yaml
uses: actions/ghcr-cleaning-action@v1
with:
  user: ${{ github.repository_owner }}
  password: ${{ secrets.YOUR_SECRET_PAT }}
  package: terraform-graph-beautifier
  repository: pcasteran/terraform-graph-beautifier
  dry-run: true
```

**The input `dry-run` is set to `true` in the sample above to let you test the behavior of the action and configure it
to your needs with no risk of deleting unwanted objects.**

## Inputs

| Name           | Type   | Required | Description                                                                                                                           |
|----------------|--------|----------|---------------------------------------------------------------------------------------------------------------------------------------|
| `registry`     | String | No       | The URL of the container registry. Defaults to `ghcr.io`.                                                                             |
| `user`         | String | No       | The container registry user. Defaults to `${{ github.repository_owner }}`.                                                            |
| `password`     | String | Yes      | The container registry user password or access token. See the [authentication](#authentication) section                               |
| `package`      | String | Yes      | The name of the package to clean.                                                                                                     |
| `repository`   | String | No       | The GitHub repository (format owner/repository) in which to check the pull requests statuses. Defaults to `${{ github.repository }}`. |
| `pr-tag-regex` | String | No       | The regular expression used to match the pull request tags, must include one capture group for the PR id. Defaults to `^pr-(\\d+).*`. |
| `dry-run`      | Bool   | No       | If true, compute everything but do no perform the deletion. Defaults to `false`.                                                      |
| `debug`        | Bool   | No       | Enable the debug logs. Defaults to `false`.                                                                                           |

## Outputs

This action does not output any value.

## Authentication

As per
the [documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-with-a-personal-access-token-classic),
the authentication to the GHCR registry must be done using a personal access token. Only classic tokens can be used,
fined-grained ones are currently (2023-01) not supported.

The [recommendation](https://docs.github.com/en/rest/packages?apiVersion=2022-11-28#delete-package-version-for-a-user)
is to create a new PAT with only the `read:packages` and `delete:packages` scopes. To do so, you can
use [this](https://github.com/settings/tokens/new?scopes=read:packages,delete:packages) link.
