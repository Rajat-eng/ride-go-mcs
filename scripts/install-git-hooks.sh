#!/usr/bin/env bash

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"

git config core.hooksPath .githooks
chmod +x "$repo_root/.githooks/pre-commit"

echo "Installed git hooks from .githooks/"
echo "Current hooks path: $(git config --get core.hooksPath)"
