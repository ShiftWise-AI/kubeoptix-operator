# Contributing

This repository is one component of the KubeOptix platform. Use the same branch flow in each component repository.

## Branch strategy

- `develop` is the integration branch for normal work.
- `stage` is the release-candidate branch and accepts pull requests only from `develop`.
- `main` is the production/release branch. It accepts pull requests from `stage` or `hotfix/*`.
- Create planned features and ordinary fixes as `feature/<short-description>` from `develop`.
- Create production fixes as `hotfix/<short-description>` from the latest `main`.

GitHub records commit ancestry, not the branch a contributor selected when creating a branch. The workflow checks PR names and destinations; creating a branch from the required base remains a contributor responsibility.

## Feature flow

```text
feature/* -> develop -> stage -> main
```

Open a PR from `feature/*` only to `develop`. Promote a release through a PR from `develop` to `stage`, validate it there, then open a PR from `stage` to `main`.

## Hotfix flow

```text
hotfix/* -> main
hotfix/* -> develop -> stage
```

Create the hotfix from `main` and open PRs from that branch to both `main` and `develop`. Do not open a hotfix PR directly to `stage`; stage accepts only `develop`. Use merge commits for both hotfix PRs, keep the hotfix branch until both are merged, and avoid cherry-picking the same fix into separate commits. Promote `develop` to `stage` through the normal release PR. Review that PR carefully: it includes all changes on `develop` that are not yet on `stage`, not only the hotfix.

## Pull requests and protections

- Use the required source and target branches above and describe the change, impact, and validation performed.
- Run the checks documented by the component README before requesting a merge.
- The `Validate Branch Flow / check-flow` workflow rejects invalid PR destinations.
- Repository administrators must enable GitHub rulesets or branch protection for `main`, `stage`, and `develop`, requiring PRs and the `Validate Branch Flow / check-flow` status check, blocking direct updates and deletion, and preventing bypass. The workflow alone does not block direct pushes or branch deletion.
- No minimum review count is prescribed by this guide.

## Release process

1. Open `develop` -> `stage` for the release candidate.
2. Validate the candidate in the stage environment and resolve release-blocking findings.
3. Open `stage` -> `main` and merge after the release is approved.
4. Tag the released commit on `main` as `vMAJOR.MINOR.PATCH` and publish the corresponding GitHub release.

## Commands

Start a feature from the current `develop`:

```bash
git fetch origin
git switch develop
git pull --ff-only origin develop
git switch -c feature/<short-description>
git push --set-upstream origin feature/<short-description>
```

Start a hotfix from the current `main`:

```bash
git fetch origin
git switch main
git pull --ff-only origin main
git switch -c hotfix/<short-description>
git push --set-upstream origin hotfix/<short-description>
```

Open the PRs in the flows above; do not push commits directly to `main`, `stage`, or `develop`.