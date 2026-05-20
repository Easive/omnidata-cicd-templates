# Using Omnidata CI/CD Templates

This repository provides reusable GitHub Actions workflows for all Omnidata microservices. This guide explains how to integrate these workflows into a service repository.

## Quick Start

1. Copy `templates/service-ci.yml` to `.github/workflows/ci.yml` in the service repository.
2. Replace all occurrences of `SERVICE_NAME` with the actual service name (e.g., `omnidata-schema-worker`).
3. Copy `templates/Dockerfile.go` to the root of the service repository and adjust `./cmd/server` if necessary.
4. Copy `templates/.golangci.yml` to the root of the service repository.
5. Copy `templates/dependabot.yml` to `.github/dependabot.yml` in the service repository.
6. Configure the required secrets in the repository or organization settings (see below).

## Required Secrets

| Secret | Scope | Description |
|---|---|---|
| `KUBECONFIG_STAGING` | Repository or org | Base64-encoded kubeconfig for the staging cluster |
| `KUBECONFIG_PRODUCTION` | Repository or org | Base64-encoded kubeconfig for the production cluster |

`GITHUB_TOKEN` is provided automatically by GitHub Actions and does not need to be configured.

To encode your kubeconfig:
```bash
base64 -w 0 ~/.kube/your-cluster-config.yaml
```

## Available Reusable Workflows

All workflows live in `.github/workflows/` of this repository and are referenced with `@main`.

### `lint-and-test.yml`

Runs golangci-lint and Go unit tests with race detection and coverage.

```yaml
jobs:
  lint-and-test:
    uses: easive/omnidata-cicd-templates/.github/workflows/lint-and-test.yml@main
    with:
      go-version: '1.22'          # optional, default: '1.22'
      working-directory: '.'      # optional, default: '.'
```

Outputs: uploads `coverage.out` as a build artifact named `coverage-report`.

### `integration-test.yml`

Starts Docker Compose test dependencies, runs integration-tagged tests, then tears down.

```yaml
jobs:
  integration-test:
    uses: easive/omnidata-cicd-templates/.github/workflows/integration-test.yml@main
    with:
      go-version: '1.22'
      compose-file: 'docker-compose.test.yml'   # optional
      working-directory: '.'                     # optional
```

The service repository must have a `docker-compose.test.yml` (or the file specified via `compose-file`) that defines test dependencies (databases, message brokers, etc.) with healthchecks.

### `build-and-publish.yml`

Builds a multi-platform Docker image (`linux/amd64`, `linux/arm64`) and pushes it to GHCR.

```yaml
jobs:
  build:
    uses: easive/omnidata-cicd-templates/.github/workflows/build-and-publish.yml@main
    with:
      service-name: omnidata-schema-worker    # required
      go-version: '1.22'                      # optional
      image-name: ''                          # optional, defaults to ghcr.io/easive/{service-name}
    permissions:
      contents: read
      packages: write
```

**Tag strategy:**
- Every build: `ghcr.io/easive/{service}:sha-{short-sha}`
- On `v*` tags: additionally `ghcr.io/easive/{service}:{semver}` and `ghcr.io/easive/{service}:latest`

### `deploy.yml`

Updates the image tag via Kustomize and applies manifests to the target Kubernetes cluster.

```yaml
jobs:
  deploy:
    uses: easive/omnidata-cicd-templates/.github/workflows/deploy.yml@main
    with:
      environment: staging          # required: staging | production
      service-name: omnidata-schema-worker   # required
      image-tag: sha-abc1234        # required
      kube-namespace: omnidata      # optional, default: omnidata
    secrets:
      KUBECONFIG_STAGING: ${{ secrets.KUBECONFIG_STAGING }}
      KUBECONFIG_PRODUCTION: ${{ secrets.KUBECONFIG_PRODUCTION }}
```

The service repository must have a `k8s/` directory with a valid `kustomization.yaml`.

### `security-scan.yml`

Runs `govulncheck` on Go source code. Optionally runs Trivy against a built Docker image, failing on CRITICAL vulnerabilities.

```yaml
jobs:
  security:
    uses: easive/omnidata-cicd-templates/.github/workflows/security-scan.yml@main
    with:
      go-version: '1.22'
      image-name: 'ghcr.io/easive/omnidata-schema-worker:sha-abc1234'  # optional
```

### `pr-checks.yml`

Composes `lint-and-test` and `security-scan` (govulncheck only) in parallel. Intended for use in pull request checks.

```yaml
jobs:
  pr-checks:
    uses: easive/omnidata-cicd-templates/.github/workflows/pr-checks.yml@main
    with:
      go-version: '1.22'
      working-directory: '.'
```

### `release.yml`

Orchestrates a full release: builds and publishes the image, then deploys to the target environment. Intended for use on `v*` tag pushes.

```yaml
jobs:
  release:
    uses: easive/omnidata-cicd-templates/.github/workflows/release.yml@main
    with:
      service-name: omnidata-schema-worker   # required
      environment: production                # optional, default: production
      go-version: '1.22'                    # optional
    secrets:
      KUBECONFIG_STAGING: ${{ secrets.KUBECONFIG_STAGING }}
      KUBECONFIG_PRODUCTION: ${{ secrets.KUBECONFIG_PRODUCTION }}
```

## Pin to a Specific Version

For production stability, pin to a release tag instead of `@main`:

```yaml
uses: easive/omnidata-cicd-templates/.github/workflows/lint-and-test.yml@v1.2.0
```

Check the [releases page](https://github.com/easive/omnidata-cicd-templates/releases) for available tags.

## Kubernetes Requirements

Each service repository must have:

```
k8s/
  kustomization.yaml    # must declare the service image as a managed image
  deployment.yaml
  service.yaml
  ...
```

The `kustomization.yaml` must list the service image so Kustomize can update it:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

images:
  - name: ghcr.io/easive/omnidata-schema-worker
    newTag: latest

resources:
  - deployment.yaml
  - service.yaml
```

## Adding Integration Tests

Create a `docker-compose.test.yml` at the repository root with the dependencies needed for integration tests. All services must define a `healthcheck` so the pipeline can wait for them to be ready.

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: testdb
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U test -d testdb"]
      interval: 5s
      timeout: 5s
      retries: 10
```

Then tag your integration tests with `//go:build integration` and call the workflow:

```yaml
jobs:
  integration-test:
    uses: easive/omnidata-cicd-templates/.github/workflows/integration-test.yml@main
    with:
      compose-file: docker-compose.test.yml
```
