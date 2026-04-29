# Release Process

This document describes how to create a new release of obs-mcp.

## Prerequisites

- A GPG key configured for [signing commits and tags](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits) (`git config user.signingkey`)
- Push access to the repository

## Branch management and versioning strategy

We use [Semantic Versioning](https://semver.org/) and maintain a separate branch for each minor release, named `release-<major>.<minor>` (e.g., `release-0.1`, `release-0.2`).

### Flow

- New features and changes are merged into `main`.
- When ready to cut a new minor release, create a `release-X.Y` branch from `main`. From this point on, all release work (changelog, tagging) happens on the release branch.
- Bug fixes for a released version are merged into the **latest release branch**.
- Bug fixes from the release branch are then merged back into `main` so that `main` always contains all commits from the latest release branch.

```
main:          A---B---C-----------G
                        \         /
release-0.1:             C'--D--fix1--fix2
                          |        |
                        v0.1.0   v0.1.1
```

### Rules

- `main` should always contain all commits from the latest release branch.
- If a bug fix is accidentally merged into `main` instead of the release branch, cherry-pick the commits into the release branch and merge back into `main`. Avoid this situation when possible.
- Maintaining release branches for older minor releases happens on a best effort basis.

> [!NOTE]
> Pushing to a release branch does not trigger the release workflow — only tag pushes (`v*`) trigger it. CI checks (lint, unit tests, e2e) will run as usual.

## How to cut a new release

### New minor release

For a new minor release, work from `main`. For a patch release, see [Patch release](#patch-release).

#### 1. Create the release branch

Ensure `main` is up to date:

```bash
git checkout main
git pull <remote> main --rebase
```

Replace `<remote>` with the name of your upstream remote (i.e., the one pointing to `github.com/rhobs/obs-mcp`). Verify with `git remote -v`.

Create the release branch from `main`. You can do this from the GitHub UI (create a new branch from `main` named `release-X.Y`) or from the command line:

```bash
git checkout -b release-X.Y
git push <remote> release-X.Y
```

From this point on, all release work happens on a branch based off `release-X.Y`.

#### Pre-releases (optional)

Before cutting a stable release, you can optionally tag a release candidate from the release branch to test artifacts and get early feedback. No changelog update is needed for pre-releases — keep the `[Unreleased]` section updated as changes land, and it will be promoted to a versioned section during the stable release.

```bash
git checkout release-X.Y
export VERSION=X.Y.Z-rc.N
export TAG="v${VERSION}"
make tag VERSION=${VERSION}
git push <remote> ${TAG}
```

Pre-releases are marked as "pre-release" on GitHub and won't be considered the "latest" release. Use them to:

- Test release artifacts before a stable release
- Get feedback from early adopters
- Verify the release process

#### 2. Update CHANGELOG.md

Create a branch from the release branch, add a new section following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format:

```bash
git fetch <remote> release-X.Y
git checkout -b cut-vX.Y.Z <remote>/release-X.Y
```

```markdown
## [X.Y.Z]

### Added
- New feature description

### Changed
- Change description

### Fixed
- Bug fix description
```

Commit and push to your fork:

```bash
git add CHANGELOG.md
git commit -s -S -m "docs: update changelog for vX.Y.Z"
git push <fork> cut-vX.Y.Z
```

Open a PR targeting `release-X.Y`, review, and merge.

Pull the merged changes to your local release branch:

```bash
git checkout release-X.Y
git pull <remote> release-X.Y --rebase
```

Verify the branch points to the expected commit (the merged changelog PR):

```bash
git log --oneline -5
```

#### 3. Create and push the tag

Set the version and create a signed tag:

```bash
export VERSION=X.Y.Z
export TAG="v${VERSION}"
make tag VERSION=${VERSION}
```

Verify the tag:

```bash
git verify-tag ${TAG}
git log --oneline -5  # confirm the tag points to the expected commit
```

Push the tag:

```bash
git push <remote> ${TAG}
```

Pushing the tag triggers the [release workflow](.github/workflows/release.yaml), which:

- Runs unit tests
- Builds cross-platform binaries (linux/darwin, amd64/arm64) via [GoReleaser](.goreleaser.yaml)
- Signs release archives with [cosign](https://docs.sigstore.dev/quickstart/quickstart-ci/) (keyless)
- Creates a GitHub release with the binaries, checksums, and auto-generated changelog

#### 4. Verify the release

- Check the [Actions tab](../../actions/workflows/release.yaml) for the workflow run
- Confirm the release appears under [Releases](../../releases) with the expected assets:
  - `obs-mcp_<version>_linux_amd64.tar.gz`
  - `obs-mcp_<version>_linux_arm64.tar.gz`
  - `obs-mcp_<version>_darwin_amd64.tar.gz`
  - `obs-mcp_<version>_darwin_arm64.tar.gz`
  - `checksums.txt`
  - `.bundle` signature files for each archive

#### 5. Merge back to main

Create a PR to merge the release branch back into `main` to ensure `main` contains the changelog and any release commits:

```bash
git checkout -b merge-release-X.Y release-X.Y
git pull <remote> main --rebase
git push <fork> merge-release-X.Y
```

Open a PR targeting `main`, review, and merge.

### Patch release

For patch releases, work on the existing release branch:

```bash
git fetch <remote> release-X.Y
git checkout -b <bugfix-branch> <remote>/release-X.Y
# make your fix changes
git add . && git commit -s -S -m "fix: description of the fix"
git push <fork> <bugfix-branch>
```

Open a PR targeting `release-X.Y`, review, and merge.

Then follow the same steps as a new minor release starting from [Update CHANGELOG.md](#2-update-changelogmd) to update the changelog, tag, verify, and merge back to `main`.

## Manual release (via workflow dispatch)

A release can also be triggered manually from the GitHub Actions UI:

1. Go to **Actions** > **release** workflow
2. Click **Run workflow**
3. Enter the tag (e.g., `v0.1.0`) and run

## Verifying release signatures

All release artifacts are signed using [cosign](https://github.com/sigstore/cosign) with keyless signing (via GitHub OIDC). Signatures and certificates are stored in bundle files for simplified verification.

```bash
# Download artifacts
wget https://github.com/rhobs/obs-mcp/releases/download/v<version>/obs-mcp_<version>_<os>_<arch>.tar.gz
wget https://github.com/rhobs/obs-mcp/releases/download/v<version>/obs-mcp_<version>_<os>_<arch>.tar.gz.bundle

# Verify using bundle
cosign verify-blob \
  --bundle obs-mcp_<version>_<os>_<arch>.tar.gz.bundle \
  --certificate-identity-regexp 'https://github.com/rhobs/obs-mcp' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  obs-mcp_<version>_<os>_<arch>.tar.gz
```

The bundle file contains both the signature and certificate, making verification simpler compared to the older separate `.sig` and `.pem` files.

## Versioning guidelines

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (X.0.0): Incompatible API changes
- **MINOR** (x.Y.0): New functionality, backwards compatible
- **PATCH** (x.y.Z): Bug fixes, backwards compatible

### Examples

- `v0.1.0` - Initial release
- `v0.2.0` - Added new tools or features
- `v0.2.1` - Bug fixes
- `v1.0.0` - First stable release
- `v1.0.0-rc.1` - Release candidate for v1.0.0

## Local testing

To test the release process locally without publishing:

```bash
goreleaser release --snapshot --clean
```

Built artifacts will be in the `dist/` directory.
