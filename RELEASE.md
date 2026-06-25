# Release Process

Releases are managed with GoReleaser. You only need to update the `version` file. When that change is pushed to `main`, GitHub Actions creates the matching tag and publishes the release.

## What gets released

For each version, GoReleaser publishes:

- A GitHub Release
- Linux, macOS, and Windows binaries
- Compressed archives
- `SHASUMS256.txt`
- Stable Docker images on Docker Hub

Docker images are published as:

```text
<dockerhub-user>/i18n-mcp:<version>
<dockerhub-user>/i18n-mcp:latest
```

## Creating a release

Update the `version` file:

```sh
echo "0.4.0" > version
```

Commit and push the change to `main`:

```sh
git add version
git commit -m "bump version to 0.4.0"
git push origin main
```

That is enough. The tag workflow will create the `v0.4.0` tag automatically. The tag then triggers the release workflow, which runs GoReleaser.

The `version` file must contain a semantic version without the leading `v`:

```text
0.4.0
```

The generated Git tag will include the leading `v`:

```text
v0.4.0
```

## Workflows

There are two release-related workflows:

```text
.github/workflows/tag.yml      # creates vX.Y.Z when version changes on main
.github/workflows/release.yml  # publishes releases from v* tags
```

Before publishing, the release workflow checks that:

1. The tag matches the `version` file.
2. `go mod tidy` does not change `go.mod` or `go.sum`.
3. `go vet ./...` passes.
4. `go test ./...` passes.

If those checks pass, GoReleaser creates the GitHub Release, publishes Docker images, generates SBOMs, and creates artifact attestations.

## Manual tag releases

Manual tag releases are still supported.

If needed, maintainers can push a tag directly:

```sh
git tag -a v0.4.0 -m "v0.4.0"
git push origin v0.4.0
```

The tag must match the `version` file, otherwise the workflow fails.

## Local checks

To build a local snapshot without publishing anything:

```sh
mise run release-snapshot
```

Snapshot artifacts are written to `dist/`, which is ignored by Git.

## Release notes

GoReleaser generates release notes from commit messages since the previous tag.

Use clear commit messages, for example:

```text
feat: add locale summary output
fix: handle missing config file
docs: document release process
```

The generated notes are grouped into:

- Added
- Fixed
- Documentation
- Other
