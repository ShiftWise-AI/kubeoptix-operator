# Contributing

This repository is one component of the KubeOptix platform.

## Branches

- Keep only `develop` and `main`.
- Start changes from `develop`; `main` is the production/release branch.
- Describe the change, impact, and validation when proposing a merge from `develop` to `main`.

## Validation and releases

Run the checks documented in the component README manually. Image builds, publication, and deployment are maintainer-managed manual operations.

## Commands

```bash
git fetch origin
git switch develop
git pull --ff-only origin develop
```