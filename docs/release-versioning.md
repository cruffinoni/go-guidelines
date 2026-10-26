# Go binary versioning and releases

This document defines releases for the Go `gg-lint` binary. Git tags are the source of truth for binary releases.

## Version classes

Every unique, valid version tag pushed to `origin` triggers a GitHub Release after the release workflow's validation and test gates pass.

- **Stable:** `vMAJOR.MINOR.PATCH` with no hyphen suffix, e.g. `v2.4.0`. It is published as a normal release and is eligible to become GitHub's latest release.
- **Prerelease / unstable:** `vMAJOR.MINOR.PATCH-<label>.<counter>`, e.g. `v2.4.0-alpha.1`, `v2.4.0-alpha.2`, `v2.4.0-beta.1`, `v2.4.0-rc.1`, or `v2.4.0-dev.1`. Each is a distinct released version, but none is stable; it must be marked as a GitHub prerelease and must not replace the latest stable release.
- Tags outside the supported grammar are rejected by the release workflow. A commit push without a new version tag does not publish a release.

The label comes before the counter: both `v2.4.0-alpha.1` and `v2.4.0-alpha.2` are prereleases of the `2.4.0` line. The stable promotion is `v2.4.0`, not `v2.4.1`.

## Lifecycle and manual tag creation

The repository no longer ships a local tag-bump helper. Choose a valid, unused tag manually; the release workflow validates new tags and publishes only after all required checks pass.

For example, after stable `v2.4.0`, create and push the first alpha tag for the next patch:

```bash
git tag --list 'v*' --sort=-version:refname
git tag -a v2.4.1-alpha.1 -m "Release v2.4.1-alpha.1"
git push origin HEAD
git push origin v2.4.1-alpha.1
```

Each prerelease tag is a distinct version. Continue the same alpha series with `v2.4.1-alpha.2`; changing the label on the same core starts the new label at `.1`, e.g. `v2.4.1-beta.1`.

Promote the latest prerelease core to stable by creating its unsuffixed tag. For example:

```bash
git tag -a v2.4.1 -m "Release v2.4.1"
git push origin HEAD
git push origin v2.4.1
```

This promotes `v2.4.1-rc.1` to `v2.4.1`; it does not increment the core. After stable `v2.4.1`, the next patch, minor, or major prerelease should use the corresponding next core. Never reuse a published tag; if a release fails, repair the workflow/configuration and use a new version.

## CI and publishing behavior

`.github/workflows/release.yml` is tag-triggered. Its first job validates the exact tag grammar. Native Linux, macOS, and Windows tests and provider-neutral validation must all pass before GoReleaser publishes. The release job passes `${{ github.ref_name }}` as `GORELEASER_CURRENT_TAG`, so GoReleaser uses the tag that triggered this run even when multiple tags point to the same commit. The action uses the pinned GoReleaser version from the workflow.

`.goreleaser.yaml` sets `release.prerelease: auto`, so a tag suffix such as `-alpha.1` is published as a prerelease. Stable tags have no suffix. `release.make_latest: true` allows stable releases to become latest; GitHub does not allow a release marked prerelease to be latest. Stable tags should be published monotonically; verify the actual `latest` pointer after the first real release.

GoReleaser injects the tag version, commit, and build time into the Go CLI. The executable's `gg-lint version` output should therefore match the release tag (without the leading `v`).
