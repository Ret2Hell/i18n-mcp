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

That is enough. The release workflow will create the `v0.4.0` tag automatically and then run GoReleaser.

The `version` file must contain a semantic version without the leading `v`:

```text
0.4.0
```

The generated Git tag will include the leading `v`:

```text
v0.4.0
```

## Release workflow

The workflow lives at:

```text
.github/workflows/release.yml
```

It runs when `version` changes on `main`.

Before publishing, it checks that:

1. The version is valid.
2. The generated tag does not already point to a different commit.
3. `go mod tidy` does not change `go.mod` or `go.sum`.
4. `go vet ./...` passes.
5. `go test ./...` passes.

If those checks pass, GoReleaser creates the GitHub Release and publishes Docker images.

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
