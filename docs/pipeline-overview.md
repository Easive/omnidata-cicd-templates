# Pipeline Overview

This document describes the complete CI/CD pipeline for Omnidata microservices.

## Pipeline Triggers

| Event | Branches / Refs | Pipeline |
|---|---|---|
| `pull_request` | `main`, `develop` | PR Checks (lint + test + govulncheck) |
| `push` | `feature/**`, `hotfix/**` | PR Checks only |
| `push` | `develop` | PR Checks → Build & Publish → Deploy Staging |
| `push` | `main` | PR Checks → Build & Publish (sha tag only) |
| `push` (tag) | `v*` | PR Checks → Build & Publish → Deploy Production |

## Full Pipeline Diagram

```
─────────────────────────────────────────────────────────────────────────────
  PULL REQUEST / feature / hotfix push
─────────────────────────────────────────────────────────────────────────────

  ┌─────────────────────────────────────────────────────────────────────┐
  │                          pr-checks.yml                              │
  │                                                                     │
  │  ┌──────────────────────────┐   ┌─────────────────────────────────┐ │
  │  │    lint-and-test.yml     │   │       security-scan.yml         │ │
  │  │                          │   │                                 │ │
  │  │  1. checkout             │   │  1. checkout                    │ │
  │  │  2. setup-go + cache     │   │  2. setup-go + cache            │ │
  │  │  3. golangci-lint run    │   │  3. govulncheck ./...           │ │
  │  │  4. go test -race \      │   │                                 │ │
  │  │       -coverprofile=...  │   │  (Trivy skipped: no image yet)  │ │
  │  │  5. upload coverage.out  │   │                                 │ │
  │  └──────────────────────────┘   └─────────────────────────────────┘ │
  │              (parallel)                      (parallel)              │
  └─────────────────────────────────────────────────────────────────────┘


─────────────────────────────────────────────────────────────────────────────
  develop branch push  →  staging deploy
─────────────────────────────────────────────────────────────────────────────

  ┌──────────────┐     ┌──────────────────────────┐     ┌──────────────────┐
  │  pr-checks   │────▶│   build-and-publish.yml  │────▶│   deploy.yml     │
  │  (see above) │     │                          │     │  environment:    │
  └──────────────┘     │  1. setup QEMU + buildx  │     │    staging       │
                       │  2. login GHCR           │     │                  │
                       │  3. extract tags:        │     │  1. decode       │
                       │     sha-{short-sha}      │     │     KUBECONFIG   │
                       │  4. docker buildx build  │     │  2. kustomize    │
                       │     --push               │     │     edit set     │
                       │     --platform           │     │     image        │
                       │     linux/amd64,arm64    │     │  3. kubectl      │
                       │                          │     │     apply -k k8s/│
                       │  Image pushed:           │     │  4. kubectl      │
                       │  ghcr.io/easive/         │     │     rollout      │
                       │    {svc}:sha-{sha}       │     │     status       │
                       └──────────────────────────┘     │     --timeout    │
                                                        │     300s         │
                                                        └──────────────────┘


─────────────────────────────────────────────────────────────────────────────
  v* tag push  →  production release  (via release.yml)
─────────────────────────────────────────────────────────────────────────────

  ┌──────────────┐     ┌──────────────────────────┐     ┌──────────────────┐
  │  pr-checks   │────▶│   build-and-publish.yml  │────▶│   deploy.yml     │
  │  (see above) │     │                          │     │  environment:    │
  └──────────────┘     │  Tags pushed:            │     │    production    │
                       │  - sha-{short-sha}       │     │                  │
                       │  - {semver} (e.g. 1.4.2) │     │  image-tag:      │
                       │  - {major}.{minor}       │     │    {tag-name}    │
                       │  - latest                │     │  (e.g. v1.4.2)   │
                       └──────────────────────────┘     └──────────────────┘


─────────────────────────────────────────────────────────────────────────────
  Optional: integration-test.yml  (called explicitly by service CI)
─────────────────────────────────────────────────────────────────────────────

  ┌──────────────────────────────────────────────────────────────────────┐
  │                       integration-test.yml                           │
  │                                                                      │
  │  1. checkout + setup-go + cache                                      │
  │  2. docker compose -f docker-compose.test.yml up -d --wait           │
  │  3. wait for all healthchecks (timeout 60s)                          │
  │  4. go test -tags integration -timeout 5m ./...                      │
  │  5. docker compose down --volumes --remove-orphans   (always)        │
  └──────────────────────────────────────────────────────────────────────┘


─────────────────────────────────────────────────────────────────────────────
  Optional: security-scan.yml with image (post-build, scheduled, etc.)
─────────────────────────────────────────────────────────────────────────────

  ┌──────────────────────────────────────────────────────────────────────┐
  │                         security-scan.yml                            │
  │                                                                      │
  │  ┌──────────────────────────┐   ┌─────────────────────────────────┐  │
  │  │     govulncheck job      │   │      trivy-image-scan job       │  │
  │  │                          │   │   (only if image-name provided) │  │
  │  │  govulncheck ./...       │   │                                 │  │
  │  │                          │   │  trivy image scan               │  │
  │  │                          │   │  severity: CRITICAL             │  │
  │  │                          │   │  exit-code: 1 (fail on CRIT)   │  │
  │  └──────────────────────────┘   └─────────────────────────────────┘  │
  │              (parallel)                      (parallel)               │
  └──────────────────────────────────────────────────────────────────────┘
```

## Workflow Composition Map

```
service CI (service-ci.yml)
│
├── pr-checks.yml
│   ├── lint-and-test.yml
│   └── security-scan.yml  (govulncheck only)
│
├── build-and-publish.yml  (main, develop, v* tag)
│
├── deploy.yml  (develop → staging)
│   environment: staging
│
└── deploy.yml  (v* tag → production)
    environment: production

release.yml  (standalone orchestrator for v* tag)
├── build-and-publish.yml
└── deploy.yml
```

## Image Tagging Summary

| Trigger | Tags produced |
|---|---|
| Any push / PR | `sha-{7-char-sha}` |
| `v*` tag (e.g. `v1.4.2`) | `sha-{sha}`, `1.4.2`, `1.4`, `latest` |

## Artifact Retention

| Artifact | Retention |
|---|---|
| `coverage-report` (coverage.out) | 7 days |
| `trivy-scan-results` | 14 days |
