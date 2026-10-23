# Contributing

This repository is one component of the KubeOptix platform.

## Branches

- Keep only `develop` and `main`.
- Start changes from `develop`; `main` is the production/release branch.
- Describe the change, impact, and validation when proposing a merge from `develop` to `main`.

## Validation and releases

Run the checks documented in the README before proposing changes. GitHub Actions runs Go and release-automation tests on pull requests. Merges into `main` automatically reserve the next patch version and publish scanned Operator, bundle, and catalog images to Quay through the `quay-release` environment. Do not create release tags manually. Failed releases can be resumed with the original merge SHA without incrementing the version again. Deployment to a cluster remains a separate maintainer operation.

## Commands

```bash
git fetch origin
git switch develop
git pull --ff-only origin develop
```