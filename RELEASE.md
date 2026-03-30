# Release Process

This document describes how to create a new release of obs-mcp.

## Prerequisites

- A GPG key configured for signing git tags (`git config user.signingkey`)
- Push access to the repository

## Steps

### 1. Update CHANGELOG.md

Ensure main is up to date:

```bash
git checkout main
git pull <remote> main --rebase
```

Replace `<remote>` with the name of your upstream remote. Verify with `git remote -v`.

Create a branch, add a new section following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format:

```bash
git checkout -b release-vX.Y.Z
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
git commit -m "docs: update changelog for vX.Y.Z"
git push <fork> release-vX.Y.Z
```

Open a PR from your fork to upstream `main` and merge.

### 2. Create and push the tag

Pull the merged changelog into main:

```bash
git checkout main
git pull <remote> main --rebase
```

Verify tests pass:

```bash
make test-unit
make lint
```

Set the version and create a signed tag:

```bash
export VERSION=0.1.0
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

### 3. Verify the release

- Check the [Actions tab](../../actions/workflows/release.yaml) for the workflow run
- Confirm the release appears under [Releases](../../releases) with the expected assets:
  - `obs-mcp_<version>_linux_amd64.tar.gz`
  - `obs-mcp_<version>_linux_arm64.tar.gz`
  - `obs-mcp_<version>_darwin_amd64.tar.gz`
  - `obs-mcp_<version>_darwin_arm64.tar.gz`
  - `checksums.txt`
  - `.bundle` signature files for each archive

## Manual release (via workflow dispatch)

A release can also be triggered manually from the GitHub Actions UI:

1. Go to **Actions** > **release** workflow
2. Click **Run workflow**
3. Enter the tag (e.g., `v0.1.0`) and run

## Pre-releases

Pre-releases follow the same process as stable releases but use the tag format `vX.Y.Z-rc.N`. No changelog PR is needed at release time keep the `[Unreleased]` section updated as changes land in main, and it will be promoted to a versioned section during the stable release.

```bash
git checkout main
git pull <remote> main --rebase
export VERSION=0.1.0-rc.1
export TAG="v${VERSION}"
make tag VERSION=${VERSION}
git push <remote> ${TAG}
```

Pre-releases are marked as "pre-release" on GitHub and won't be considered the "latest" release. Use them to:

- Test release artifacts before a stable release
- Get feedback from early adopters
- Verify the release process

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
