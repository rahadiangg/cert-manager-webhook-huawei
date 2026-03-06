# GitHub Actions CI/CD

This repository uses GitHub Actions for continuous integration and deployment.

## Workflows

### CI (`.github/workflows/ci.yaml`)

Runs on every push and pull request to `main` and `develop` branches:

- **Test**: Runs Go tests with race detection and coverage
- **Build**: Builds Docker images for `linux/amd64` and `linux/arm64` (no push)
- **Lint**: Runs `golangci-lint` for code quality checks

### Release (`.github/workflows/release.yaml`)

Triggered by semantic version tags (e.g., `v0.12.1`):

- Strips the `v` prefix for Docker tags (e.g., `0.12.1`)
- Builds and pushes multi-arch images to GitHub Container Registry
- Creates the following tags:
  - `0.12.1` (exact version)
  - `0.12` (minor version)
  - `0` (major version)
  - `latest` (only from default branch)

## Creating a Release

### Step 1: Tag your commit

```bash
# Create and push a semantic version tag
git tag v0.1.0
git push origin v0.1.0
```

### Step 2: GitHub Actions builds and pushes

The workflow will automatically:
- Build for `linux/amd64` and `linux/arm64`
- Push to `ghcr.io/<username>/cert-manager-webhook-huawei`
- Tag as `0.1.0`, `0.1`, `0`, and `latest`

### Step 3: Use the image

```bash
docker pull ghcr.io/<username>/cert-manager-webhook-huawei:0.1.0
```

## Helm Chart Values

Update your Helm values to use the new image:

```yaml
image:
  repository: ghcr.io/<username>/cert-manager-webhook-huawei
  tag: "0.1.0"  # Without 'v' prefix
```

## Registry Permissions

Make sure your repository has packages enabled:

1. Go to **Settings** → **Actions** → **General**
2. Under **Workflow permissions**, enable **Read and write permissions**

## Authentication

The workflow uses `GITHUB_TOKEN` for authentication - no secrets needed!

## Example Workflow Run

When you push `v0.12.1`:

```bash
git tag v0.12.1
git push origin v0.12.1
```

Result:
- Docker image: `ghcr.io/<username>/cert-manager-webhook-huawei:0.12.1`
- Additional tags: `0.12`, `0`, `latest`
- Platforms: `linux/amd64`, `linux/arm64`

## Dependabot

Dependabot is configured to automatically update:
- GitHub Actions (weekly)
- Go modules (weekly)

See [`.github/dependabot.yaml`](./dependabot.yaml).
