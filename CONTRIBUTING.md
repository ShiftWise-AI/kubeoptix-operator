# Contributing

This repository is one component of the KubeOptix platform. Use the same branch flow in each component repository.

## Branch strategy

- `develop` is the integration branch for normal work.
- `main` is the production/release branch and accepts pull requests only from this repository's `develop`.
- Create planned features and ordinary fixes as `feature/<short-description>` from `develop`.
- Production fixes follow the same `feature/*` flow. There is no `stage` branch in the pipeline.

GitHub records commit ancestry, not the branch a contributor selected when creating a branch. The workflow checks PR names and destinations; creating a branch from the required base remains a contributor responsibility.

## Feature flow

```text
feature/* -> develop -> main
```

Open a PR from `feature/*` only to `develop`, then promote through a PR from `develop` to `main`. PRs run Semgrep SAST and Trivy dependency/secret scans, without compilation, tests, or container builds. Any security finding or scanner failure blocks the required `ci-required` check and merge.

## Pull requests and protections

- Use the required source and target branches above and describe the change, impact, and validation performed.
- Run the checks documented by the component README before requesting a merge.
- The `Validate Branch Flow / check-flow` workflow rejects invalid PR destinations.
- Protect `main` and `develop`, requiring PRs and the `check-flow` and `ci-required` checks, blocking direct updates and deletion, and preventing administrator bypass. The workflow alone does not enforce repository protection.
- No minimum review count is prescribed by this guide.

## Release process

1. Open `develop` -> `main` and merge after review and successful security checks.
2. The push to `main` automatically runs tests, builds and scans the image, and publishes the scanned image to Quay as `latest` and `sha-<commit>`. Failures block publication.
3. Optional `vMAJOR.MINOR.PATCH` tags must point to commits on `main`; the existing versioned-release pipeline is preserved.

## Commands

Start a feature from the current `develop`:

```bash
git fetch origin
git switch develop
git pull --ff-only origin develop
git switch -c feature/<short-description>
git push --set-upstream origin feature/<short-description>
```

Open the PRs in the flow above; do not push commits directly to `main` or `develop`.