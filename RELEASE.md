# Release Process

This document describes how to create a new release of obs-mcp.

## Prerequisites

- A GPG key configured for signing git tags (`git config user.signingkey`)
- Push access to the repository

## Steps

Use these steps for a release cut from current `main` (minor/major, or a patch when `main` *is* the tip you want to ship). If you need a **patch while `main` is ahead** with unrelated commits, use [Patch releases (when `main` is ahead)](#patch-releases-when-main-is-ahead) instead.

### 1. Update CHANGELOG.md and VERSION

Ensure main is up to date:

```bash
git checkout main
git pull <remote> main --rebase
```

Replace `<remote>` with the name of your upstream remote. Verify with `git remote -v`.

Create a branch:

```bash
git checkout -b cut-vX.Y.Z
```

**Update `CHANGELOG.md`:** promote or add a versioned section following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format (typically move content from `[Unreleased]` into the new section and leave an empty `[Unreleased]`):

```markdown
## [Unreleased]

## [vX.Y.Z] - YYYY-MM-DD

### Added
- New feature description

### Changed
- Change description

### Fixed
- Bug fix description
```

**Update `VERSION`:** set the file to the release SemVer **without** the `v` prefix (for example `0.7.1`). This file is the version embedded by `make build`, `make build-linux`, and `make container`. It must match the git tag you create in step 2 (`v` + contents of `VERSION`).

```bash
echo "X.Y.Z" > VERSION
```

Commit and push to your fork:

```bash
git add CHANGELOG.md VERSION
git commit -m "chore: cut vX.Y.Z"
git push <fork> cut-vX.Y.Z
```

Open a PR from your fork to upstream `main` and merge.

### 2. Create and push the tag (GPG-signed)

Pull the merged release commit into main:

```bash
git checkout main
git pull <remote> main --rebase
```

Verify `VERSION` matches the release you intend to tag, then run tests:

```bash
cat VERSION   # e.g. 0.7.1 → tag will be v0.7.1
make test-unit
make lint
```

Create a **GPG-signed** annotated tag with the same version:

```bash
export VERSION=$(cat VERSION)
export TAG="v${VERSION}"
make tag VERSION=${VERSION}
```

Verify the tag:

```bash
git verify-tag ${TAG}
git log --oneline -5  # confirm the tag points to the expected commit
```

Push the tag to the **upstream** remote for the `rhobs` org (`rhobs/obs-mcp`), not your fork. Confirm with `git remote -v` (often named `upstream` or `origin` depending on your clone):

```bash
git push <upstream-remote> ${TAG}
# e.g. git push upstream ${TAG}
```

Pushing the tag to upstream triggers the [release workflow](.github/workflows/release.yaml), which:

- Runs unit tests
- Builds cross-platform binaries (linux/darwin, amd64/arm64) via [GoReleaser](.goreleaser.yaml)
- Signs release archives with [cosign](https://docs.sigstore.dev/quickstart/quickstart-ci/) (keyless)
- Creates a GitHub release with the binaries, checksums, and auto-generated changelog

### 3. Verify the release

- Check the [Actions tab](https://github.com/rhobs/obs-mcp/actions/workflows/release.yaml) for the workflow run
- Confirm the release appears under [Releases](https://github.com/rhobs/obs-mcp/releases) with the expected assets:
  - `obs-mcp_<version>_linux_amd64.tar.gz`
  - `obs-mcp_<version>_linux_arm64.tar.gz`
  - `obs-mcp_<version>_darwin_amd64.tar.gz`
  - `obs-mcp_<version>_darwin_arm64.tar.gz`
  - `checksums.txt`
  - `.bundle` signature files for each archive

## Patch releases (when `main` is ahead)

Use this when you need a **patch** on an already shipped minor line (for example `v0.7.2` after `v0.7.1`) but `main` has moved on with other commits you do **not** want in that patch.

The fix should already be (or also be) on `main`. You ship the patch from a **release branch**, cherry-picking only the commits required for the fix.

### 1. Create or update the release branch

Release branches are named `release-X.Y` (no `v` prefix), for example `release-0.7` for the `0.7.x` line.

Fetch upstream and check whether the branch already exists:

```bash
git fetch <upstream-remote>
git branch -r | grep "release-X.Y" || true
```

**If the branch does not exist**, create it from the last tag on that line and push it to upstream:

```bash
# Example: patching the 0.7 line after v0.7.1
export PREV_TAG=v0.7.1
export RELEASE_BRANCH=release-0.7

git checkout -b ${RELEASE_BRANCH} ${PREV_TAG}
git push <upstream-remote> ${RELEASE_BRANCH}
```

**If the branch already exists**, check it out and update it:

```bash
git checkout ${RELEASE_BRANCH}
git pull <upstream-remote> ${RELEASE_BRANCH}
```

### 2. Cherry-pick the fix from `main`

The fix commit(s) should already be on `main`. You copy only those commits onto the release branch via a short-lived branch and a PR **whose base is `release-X.Y`**, not `main`.

Example: shipping `v0.7.2` from `release-0.7`, cherry-picking commit `abc1234` from `main`.

```bash
# Start from the release branch (already checked out / up to date from step 1)
git checkout -b cherry-pick-v0.7.2

# Find the fix on main, then cherry-pick it
git log --oneline main
git cherry-pick abc1234
# If the commit is a merge commit: git cherry-pick -m 1 <merge-sha>

# Push this branch to your fork
git push <fork> cherry-pick-v0.7.2
```

Open a pull request:

| Field | Value |
| ----- | ----- |
| **base** (merge into) | `rhobs/obs-mcp` → `release-0.7` |
| **compare** (your branch) | `<your-fork>` → `cherry-pick-v0.7.2` |

Do **not** target `main` for this PR. After it merges into `release-X.Y`, continue with the cut steps below on that release branch.

### 3. Cut CHANGELOG.md and VERSION on the release branch

After the cherry-pick PR is merged, update the release branch and open a cut PR (also targeting `release-X.Y`, not `main`):

```bash
git checkout release-X.Y
git pull <upstream-remote> release-X.Y
git checkout -b cut-vX.Y.Z
```

- Add a `## [vX.Y.Z] - YYYY-MM-DD` section to `CHANGELOG.md` describing the patch (do **not** promote unrelated `[Unreleased]` content from `main`).
- Set `VERSION` to `X.Y.Z` (no `v` prefix).

```bash
echo "X.Y.Z" > VERSION
git add CHANGELOG.md VERSION
git commit -m "chore: cut vX.Y.Z"
git push <fork> cut-vX.Y.Z
```

Open a pull request with **base** `release-X.Y` and **compare** `cut-vX.Y.Z`, then merge.

If the changelog note for the patch should also appear on `main`, open a separate PR to `main` (or rely on the fix PR’s notes under `[Unreleased]`).

### 4. Tag from the release branch (GPG-signed)

```bash
git checkout release-X.Y
git pull <upstream-remote> release-X.Y

cat VERSION
make test-unit
make lint

export VERSION=$(cat VERSION)
export TAG="v${VERSION}"
make tag VERSION=${VERSION}
git verify-tag ${TAG}
git push <upstream-remote> ${TAG}   # rhobs/obs-mcp, not your fork
```

Then verify the GitHub Release the same way as in [Steps → Verify the release](#3-verify-the-release).

### Notes

- Do **not** tag the patch from `main` if `main` contains commits beyond the release line.
- Keep `release-X.Y` around for further patches on that line; create it only once per minor line (from the previous tag) if missing.
- Minor/major releases from current `main` still follow [Steps](#steps) above (cut PR into `main`, then tag).

## Manual release (via workflow dispatch)

A release can also be triggered manually from the GitHub Actions UI:

1. Go to **Actions** > **release** workflow
2. Click **Run workflow**
3. Enter the tag (e.g., `v0.1.0`) and run

## Pre-releases

Pre-releases follow the same process as stable releases but use the tag format `vX.Y.Z-rc.N`. No changelog PR is needed at release time — keep the `[Unreleased]` section updated as changes land in main, and it will be promoted to a versioned section during the stable release.

```bash
git checkout main
git pull <upstream-remote> main --rebase
export VERSION=0.1.0-rc.1
export TAG="v${VERSION}"
make tag VERSION=${VERSION}
git push <upstream-remote> ${TAG}   # rhobs/obs-mcp, not your fork
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
