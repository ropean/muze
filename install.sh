#!/usr/bin/env bash
set -euo pipefail

# Installs a pre-built muze binary from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ropean/muze/main/install.sh | bash
#
# Pin a version (tag or "latest"):
#   MUZE_VERSION=v1.0.0 curl -fsSL ... | bash

REPO="ropean/muze"
BINARY="muze"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${MUZE_VERSION:-latest}"

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Linux)  GOOS="linux" ;;
    Darwin) GOOS="darwin" ;;
    MINGW*|MSYS*|CYGWIN*) GOOS="windows" ;;
    *) echo "Unsupported OS: $os" >&2; exit 1 ;;
  esac

  case "$arch" in
    x86_64|amd64)  GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac
}

resolve_version() {
  if [ "$VERSION" = "latest" ]; then
    VERSION="$(curl -fsSL -o /dev/null -w '%{redirect_url}' \
      "https://github.com/${REPO}/releases/latest" | grep -oE '[^/]+$')"
    if [ -z "$VERSION" ]; then
      echo "Failed to resolve latest version" >&2
      exit 1
    fi
  fi
}

download_and_install() {
  local asset_name url tmp
  asset_name="${BINARY}-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    asset_name="${asset_name}.exe"
  fi

  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset_name}"
  tmp="$(mktemp)"

  echo "Downloading ${BINARY} ${VERSION} (${GOOS}/${GOARCH})..."
  curl -fsSL -o "$tmp" "$url"
  chmod +x "$tmp"

  echo "Installing to ${INSTALL_DIR}/${BINARY}..."
  mv "$tmp" "${INSTALL_DIR}/${BINARY}"

  echo "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
}

detect_platform
resolve_version
download_and_install
