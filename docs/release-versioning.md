# Go binary versioning and releases

This document defines releases for the Go `gg-lint` binary. The repository still contains a Python package; its `pyproject.toml` and Python `__version__` are **not** the source of truth for Go binary releases and are not changed by `scripts/bump-release-version.sh`.

## Version classes

Every unique, valid version tag pushed to `origin` triggers a GitHub Release after the release workflow's validation and test gates pass.

- **Stable:** `vMAJOR.MINOR.PATCH` with no hyphen suffix, e.g. `v2.4.0`. It is published as a normal release and is eligible to become GitHub's latest release.
- **Prerelease / unstable:** `vMAJOR.MINOR.PATCH-<label>.<counter>`, e.g. `v2.4.0-alpha.1`, `v2.4.0-alpha.2`, `v2.4.0-beta.1`, `v2.4.0-rc.1`, or `v2.4.0-dev.1`. Each is a distinct released version, but none is stable; it must be marked as a GitHub prerelease and must not replace the latest stable release.
- Tags outside the supported grammar are rejected by the release workflow. A commit push without a new version tag does not publish a release.

The label comes before the counter: both `v2.4.0-alpha.1` and `v2.4.0-alpha.2` are prereleases of the `2.4.0` line. The stable promotion is `v2.4.0`, not `v2.4.1`.

## Lifecycle and bump commands

The project-owned `scripts/bump-release-version.sh` uses dry-run mode by default. It manages tags only: it never edits source files, Python package metadata, or changelogs. The shared `~/bin/bump-version.sh` is not modified or used for this release path.

From stable `v2.4.0`, start the next patch's alpha series:

```bash
scripts/bump-release-version.sh patch --pre alpha
scripts/bump-release-version.sh patch --pre alpha --commit --push
```

The first command previews; the second applies the annotated tag and pushes it. Once `v2.4.1-alpha.1` exists, the same preview/apply form with `--pre alpha` continues the same core at `v2.4.1-alpha.2`. Changing the label on the same core starts the new label at `.1`, e.g. `v2.4.1-beta.1`.

Promote the latest prerelease core to stable with:

```bash
scripts/bump-release-version.sh stable
scripts/bump-release-version.sh stable --commit --push
```

This turns `v2.4.1-rc.1` into `v2.4.1`; it does not increment the core. To start a new prerelease after stable `v2.4.1`, use `patch --pre alpha` (next patch), `minor --pre alpha`, or `major --pre alpha`.

`--commit` means apply the tag operation; the tool does not create a source commit. `--push` pushes the current `HEAD` and the new tag. The tag push triggers the release workflow. Do not push a tag that has already been published; if a release fails, repair the workflow/configuration and use a new version rather than reusing a public version tag.

## CI and publishing behavior

`.github/workflows/release.yml` is tag-triggered. Its first job validates the exact tag grammar. Native Linux, macOS, and Windows tests and provider-neutral validation must all pass before GoReleaser publishes. The release job passes `${{ github.ref_name }}` as `GORELEASER_CURRENT_TAG`, so GoReleaser uses the tag that triggered this run even when multiple tags point to the same commit. The action uses the pinned GoReleaser version from the workflow.

`.goreleaser.yaml` sets `release.prerelease: auto`, so a tag suffix such as `-alpha.1` is published as a prerelease. Stable tags have no suffix. `release.make_latest: true` allows stable releases to become latest; GitHub does not allow a release marked prerelease to be latest. Stable tags should be published monotonically; verify the actual `latest` pointer after the first real release.

GoReleaser injects the tag version, commit, and build time into the Go CLI. The executable's `gg-lint version` output should therefore match the release tag (without the leading `v`).

## Python metadata boundary

The project-owned Go release tag tool has no metadata synchronization path. The Python package remains present in the repository, but its independent release lifecycle is not defined here. Do not synchronize its version fields as a side effect of a Go binary release.
