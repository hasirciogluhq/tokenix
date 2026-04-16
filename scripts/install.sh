#!/usr/bin/env bash
set -euo pipefail

REPO="hasirciogluhq/tokenix"
BIN_NAME="tokenix"

usage() {
  cat <<EOF
Usage:
  $0 [version]

Examples:
  $0           # installs latest release
  $0 v0.1.0    # installs specific version
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

VERSION="${1:-latest}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

if [[ "$OS" != "linux" && "$OS" != "darwin" ]]; then
  echo "Unsupported operating system: $OS" >&2
  exit 1
fi

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [[ -z "$VERSION" ]]; then
    echo "Could not detect latest version from GitHub releases." >&2
    exit 1
  fi
fi

ARCHIVE="${BIN_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading ${URL}"
curl -fsSL "$URL" -o "${TMP_DIR}/${ARCHIVE}"

echo "Extracting archive"
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TARGET="${INSTALL_DIR}/${BIN_NAME}"

if [[ ! -w "$INSTALL_DIR" ]]; then
  echo "Installing to ${TARGET} with sudo"
  sudo install -m 0755 "${TMP_DIR}/${BIN_NAME}" "$TARGET"
else
  install -m 0755 "${TMP_DIR}/${BIN_NAME}" "$TARGET"
fi

echo "Installed ${BIN_NAME} at ${TARGET}"
echo "Run '${BIN_NAME} --version' to verify."
