# CI/CD Audit and Operations

## Findings (2026-10-09)

The existing workflow already triggered on main push. Run 37821792035 completed
`test` and `publish-main` successfully; tag-only `publish` being skipped on main
was expected, not evidence of missing publication. Confirmed gaps were no
security gates, mutable Actions, publication independent of GitFlow validation,
cancellable main runs, and protections requiring only `check-flow`, no approval,
and no up-to-date branch. The protection gaps affected all 21 GitFlow branches
across seven repos. Repository secrets were empty; organization secrets could
not be audited with the available GitHub token (403).

## Pipeline Contract

PRs to develop/stage/main, pushes to those branches, and merge queues run GitFlow,
Go tests, workflow syntax validation, dependency/secret scans, all three image
builds, non-root executable-image inspection, and final-image scans. The scratch
OLM bundle is data-only and is scanned for vulnerabilities/secrets, not treated
as an executable image requiring USER. `ci-required` rejects failure, skipped,
or cancelled dependencies. Trivy v0.69.3 blocks HIGH/CRITICAL including unfixed
findings. All scans precede any login/push. Actions use full SHAs, actionlint
v1.7.7 is checksum-verified, token permissions are `contents: read`, and checkout
does not persist credentials.

Publishing the exact scanned local images replaces duplicate rebuilds. Main
publishes only `shiftwise-operator:latest` and `sha-<commit>` as before. Valid
`vMAJOR.MINOR.PATCH` tags publish the versioned operator, `bundle:v<version>`,
`catalog:v<version>` and `catalog:stable`, plus SHA tags. Tags must belong to
main history and older releases cannot replace the stable catalog. Existing
catalog history, fallback loading, and `hack/generate-olm.py` remain in place.
PRs, stage/develop, and merge queues never access Quay secrets. Main runs are
not actively cancelled; GitHub can coalesce pending concurrent runs.

## Security Remediation

`x/net` and `x/text` CVEs were patched and Go checksums regenerated. Go 1.26.9
is selected in go.mod and explicitly in the UBI builder: the distribution's
Go compiler otherwise built a vulnerable 1.25.9 binary despite the toolchain
directive. The authorized scratch runtime retains a static manager, TLS trust
certificates, and numeric non-root USER. It intentionally has no shell/package
manager, removing UBI runtime packages with no available CVE fixes.

The OPM base is digest-pinned and non-root. OPM v1.74.0 and grpc-health-probe
v0.4.59 are rebuilt with patched dependencies and Go 1.26.9, replacing vulnerable
upstream binaries without changing catalog data or its runtime base. The OPM
build uses `containers_image_openpgp` for pure-Go signature support with CGO off.
File-based catalog validation was tested; legacy SQLite-based OPM commands were
not certified with this static build. Do not use this catalog runtime for legacy
SQLite catalog creation without separate validation.

## GitHub and Quay Setup

Applied and verified for main/stage/develop: require `check-flow` and
`ci-required` from GitHub Actions, up-to-date branch, at least one approval,
stale-review dismissal, last-push approval, enforce for admins, and no force
pushes/deletion. Existing checks remain. APPROVE reviews are still possible with
failed checks, but integration is blocked.

Publish changes on the existing feature branch, obtain a green PR and independent
review to develop, and promote develop -> stage -> main. PRs without the new
gate are blocked until the updated workflows run. Permit the pinned Actions in
organization policy. Protect `v*` tags with a creation ruleset limited to release
maintainers and disallow mutation/deletion. `GITHUB_TOKEN`-created pushes cannot
trigger a new workflow; use an approved GitHub App for automated release tags.

Grant a dedicated Quay robot repository Write only on:

- `quay.io/parraes/shiftwise-operator`
- `quay.io/parraes/shiftwise-operator-bundle`
- `quay.io/parraes/shiftwise-operator-catalog`

Set `QUAY_USERNAME` (full `namespace+robot`) and `QUAY_PASSWORD` (robot token)
as repository or restricted organization secrets including this GitHub repo.
No organization Admin permission is needed. Enter/rotate tokens in a secure UI
or prompt, never source, command arguments, logs, or shell tracing. Catalog
history images currently require public pull access before login. Enable Quay
vulnerability notifications as a complementary post-push control.

## Validation and Remaining Acceptance

All 14 workflows passed actionlint; aggregate failure cases and 63 GitFlow cases
passed; the 21 remote protections were re-read. Go tests and source scans passed.
Operator and catalog images built and passed strict scans with no HIGH/CRITICAL
vulnerabilities, unsafe configuration, or secrets. The static operator executed
non-root; file-based catalog validation passed non-root on a read-only filesystem,
and the rebuilt health probe confirmed the catalog gRPC service was SERVING.
Bundle generation and
a complete tagged release still need remote execution. No changed workflows
were committed, pushed, or executed remotely, and no image was published.
Acceptance requires a failing PR to remain blocked, a reviewed green promotion,
and a successful main publication with matching Quay digest. OLM releases must
also validate generated bundles, preserve upgrade history, and publish only
after every image scan succeeds.

## PR Failure Remediation (2026-10-09)

PR #39 was blocked by a newly reported x/net CVE requiring v0.60.0. The module
and rebuilt OPM catalog now use that patch, with the required transitives aligned
and checksums regenerated. Go tests passed, the main operator and catalog images
passed strict vulnerability/configuration/secret scans, and FBC validation passed
read-only. Security policy and release behavior are unchanged. The updated PR
requires a new green GitHub run and independent review; no main push or release
was performed as part of these fixes.