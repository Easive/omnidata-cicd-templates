#!/bin/bash
# Run this script ONCE as the repo owner (with workflow scope PAT or via GitHub web UI)
# to activate the reusable workflows by copying them from workflow-sources/ to workflows/
set -euo pipefail

REPO="Easive/omnidata-cicd-templates"
BRANCH="main"

echo "Activating CI/CD workflows for $REPO..."

SRC_DIR=".github/workflow-sources"
DST_DIR=".github/workflows"

mkdir -p "$DST_DIR"

for f in "$SRC_DIR"/*.yml; do
  name=$(basename "$f")
  cp "$f" "$DST_DIR/$name"
  echo "  Copied: $name"
done

echo "Done! Commit and push with a PAT that has workflow scope:"
echo "  git add .github/workflows/"
echo "  git commit -m \"chore: activate reusable workflow files\""
echo "  git push"

