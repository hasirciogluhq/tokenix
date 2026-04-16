#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <module-or-owner/repo> <version-tag>"
  echo "Example 1: $0 github.com/hasirciogluhq/tokenix v0.1.0"
  echo "Example 2: $0 hasirciogluhq/tokenix v0.1.0"
}

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

module="$1"
version="$2"

# Accept either full module path (github.com/owner/repo) or owner/repo.
if [[ "$module" != *.*/* ]]; then
  module="github.com/${module}"
fi

encoded_module="${module//\//%2F}"

echo "Refreshing Go package proxy for ${module}@${version}"

echo "-> proxy.golang.org list endpoint"
curl -fsSL "https://proxy.golang.org/${module}/@v/list" >/dev/null

echo "-> proxy.golang.org info endpoint"
curl -fsSL "https://proxy.golang.org/${module}/@v/${version}.info" >/dev/null

echo "-> sum.golang.org lookup endpoint"
curl -fsSL "https://sum.golang.org/lookup/${encoded_module}@${version}" >/dev/null

echo "Go proxy refresh completed successfully."
