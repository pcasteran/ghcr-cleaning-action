# GitHub Container registry cleaning action

GitHub action allowing to clean a GitHub Container registry by deleting the unnecessary Docker images
and [manifests](https://docs.docker.com/engine/reference/commandline/manifest/) for multi-arch images.
Unnecessary can mean:

- untagged manifests and their referenced images
- tagged manifests attached to a closed Pull Request and their referenced images
- untagged images not referenced by any manifest
