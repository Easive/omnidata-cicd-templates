# omnidata-cicd-templates

Reusable GitHub Actions pipelines for all Omnidata microservices (Go 1.22).

## Repository Contents

```
.github/
  workflow-sources/          ← Reusable workflow YAML files (see Activation below)
    lint-and-test.yml
    integration-test.yml
    build-and-publish.yml
    deploy.yml
    security-scan.yml
    pr-checks.yml
    release.yml
docs/
  usage.md                   ← How to reference workflows from service repos
  pipeline-overview.md       ← ASCII diagram of the full pipeline
templates/
  service-ci.yml             ← Copy to .github/workflows/ci.yml in each service
  Dockerfile.go              ← Multi-stage Dockerfile for Go services
  .golangci.yml              ← golangci-lint configuration
  dependabot.yml             ← Dependabot config for Go modules + Actions
scripts/
  activate-workflows.sh      ← One-time activation script (see below)
```

## Activation (One-time Setup by Repo Owner)

The workflow files are stored in `.github/workflow-sources/` because the token used
during initial setup lacked the `workflow` scope. A repo owner with a classic PAT
that has `workflow` scope (or via the GitHub web UI) needs to activate them:

### Option A — GitHub Web UI (no extra permissions needed)
1. Go to https://github.com/Easive/omnidata-cicd-templates
2. For each file in `.github/workflow-sources/`:
   - Open the file → click "Edit" (pencil icon)
   - Copy its content
   - Navigate to `.github/workflows/` → "Add file" → "Create new file"
   - Paste the content, name the file identically (e.g., `lint-and-test.yml`)
   - Commit directly to `main`

### Option B — Git CLI with workflow-scope PAT
```bash
git clone https://github.com/Easive/omnidata-cicd-templates.git
cd omnidata-cicd-templates
mkdir -p .github/workflows
cp .github/workflow-sources/*.yml .github/workflows/
git add .github/workflows/
git commit -m "chore: activate reusable workflow files"
git push  # requires PAT with `workflow` scope
```

Once `.github/workflows/` is populated, the `uses:` references in service repos
(e.g., `uses: easive/omnidata-cicd-templates/.github/workflows/pr-checks.yml@main`)
will work correctly.

## Quick Start for Service Repos

See [docs/usage.md](docs/usage.md) for the full guide. The short version:

1. Copy `templates/service-ci.yml` → `.github/workflows/ci.yml` in the service repo
2. Replace `SERVICE_NAME` with the actual service name
3. Copy `templates/Dockerfile.go`, `templates/.golangci.yml`, `templates/dependabot.yml`
4. Set secrets: `KUBECONFIG_STAGING`, `KUBECONFIG_PRODUCTION`

## Pipeline Overview

See [docs/pipeline-overview.md](docs/pipeline-overview.md) for the full ASCII diagram.

| Trigger | Pipeline |
|---|---|
| PR → `main` / `develop` | lint + test + govulncheck (parallel) |
| push `develop` | PR checks → build (sha tag) → deploy staging |
| push `main` | PR checks → build (sha tag) |
| push tag `v*` | PR checks → build (semver + latest tags) → deploy production |

## Available Reusable Workflows

| Workflow | Purpose |
|---|---|
| `lint-and-test.yml` | golangci-lint + go test -race + coverage upload |
| `integration-test.yml` | Docker Compose + integration tests |
| `build-and-publish.yml` | Multi-platform Docker build → GHCR |
| `deploy.yml` | kustomize edit + kubectl apply + rollout status |
| `security-scan.yml` | govulncheck + Trivy image scan (CRITICAL fails) |
| `pr-checks.yml` | Composes lint-and-test + security-scan in parallel |
| `release.yml` | Composes build-and-publish + deploy |
