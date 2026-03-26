# Release Process

This document describes how to create a new release of obs-mcp.

## Prerequisites

- A GPG key configured for signing git tags (`git config user.signingkey`)
- Push access to the repository

## Steps

### 1. Ensure main is up to date

```bash
git checkout main
git pull <remote> main
```

Replace `<remote>` with the name of your upstream remote (e.g., `origin`). Verify with `git remote -v`.

### 2. Verify tests pass

```bash
make test-unit
make lint
```

### 3. Set the release version

```bash
export VERSION=0.1.0
export TAG="v${VERSION}"
```

### 4. Create a signed tag

```bash
make tag VERSION=${VERSION}
```

This creates a GPG-signed tag `${TAG}` locally.

### 5. Push the tag

Verify the remote points to the correct repository:

```bash
git remote -v
```

Then push (replace `<remote>` with your remote name):

```bash
git push <remote> ${TAG}
```

Pushing the tag triggers the [release workflow](.github/workflows/release.yaml), which:

- Runs unit tests
- Builds cross-platform binaries (linux/darwin, amd64/arm64) via [GoReleaser](.goreleaser.yaml)
- Signs release archives with [cosign](https://docs.sigstore.dev/cosign/overview/) (keyless)
- Creates a GitHub release with the binaries, checksums, and auto-generated changelog

### 6. Verify the release

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

Tags with pre-release suffixes (e.g., `v0.1.0-rc.1`) are automatically marked as pre-releases on GitHub.

## Verifying release signatures

Users can verify downloaded archives using cosign:

```bash
cosign verify-blob \
  --bundle obs-mcp_<version>_<os>_<arch>.tar.gz.bundle \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp "github.com/rhobs/obs-mcp" \
  obs-mcp_<version>_<os>_<arch>.tar.gz
```

## Local testing

To test the release process locally without publishing:

```bash
goreleaser release --snapshot --clean
```

Built artifacts will be in the `dist/` directory.
