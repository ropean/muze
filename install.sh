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
DEFAULT_INSTALL_DIR="${HOME}/.local/bin"
VERSION="${MUZE_VERSION:-latest}"

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"
  echo "[detect_platform] OS=$os, ARCH=$arch"

  case "$os" in
    Linux)  GOOS="linux" ;;
    Darwin) GOOS="darwin" ;;
    MINGW*|MSYS*|CYGWIN*)
      GOOS="windows"
      DEFAULT_INSTALL_DIR="${HOME}/bin"
      ;;
    *) echo "Unsupported OS: $os" >&2; exit 1 ;;
  esac

  case "$arch" in
    x86_64|amd64)  GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac

  echo "[detect_platform] GOOS=$GOOS, GOARCH=$GOARCH"
}

resolve_version() {
  echo "[resolve_version] Requested VERSION=$VERSION"
  if [ "$VERSION" = "latest" ]; then
    local redirect_url
    # Do NOT use -L here; we need the redirect URL, not the final response
    redirect_url="$(curl -fsS -o /dev/null -w '%{redirect_url}' \
      "https://github.com/${REPO}/releases/latest")"
    echo "[resolve_version] Redirect URL: $redirect_url"
    VERSION="$(echo "$redirect_url" | grep -oE '[^/]+$')"
    if [ -z "$VERSION" ]; then
      echo "Failed to resolve latest version" >&2
      exit 1
    fi
  fi
  echo "[resolve_version] Resolved VERSION=$VERSION"
}

download_and_install() {
  local asset_name url tmp
  asset_name="${BINARY}-${GOOS}-${GOARCH}"
  if [ "$GOOS" = "windows" ]; then
    asset_name="${asset_name}.exe"
  fi

  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset_name}"
  tmp="$(mktemp)"

  mkdir -p "$INSTALL_DIR"

  echo "[download] Asset name: $asset_name"
  echo "[download] Download URL: $url"
  echo "[download] Temp file: $tmp"
  echo "[download] Install dir: $INSTALL_DIR"
  echo "[download] Final path: ${INSTALL_DIR}/${BINARY}"

  echo "Downloading ${BINARY} ${VERSION} (${GOOS}/${GOARCH})..."
  if ! curl -fsSL -o "$tmp" "$url"; then
    echo "[download] ERROR: curl failed to download from $url" >&2
    rm -f "$tmp"
    exit 1
  fi

  echo "[download] Downloaded file size: $(wc -c < "$tmp") bytes"
  if [ "$GOOS" != "windows" ]; then
    chmod +x "$tmp"
  fi

  echo "Installing to ${INSTALL_DIR}/${BINARY}..."
  if ! mv "$tmp" "${INSTALL_DIR}/${BINARY}"; then
    echo "[download] ERROR: mv failed. Do you have write permission to ${INSTALL_DIR}?" >&2
    echo "[download] Try: sudo bash install.sh  or  INSTALL_DIR=~/.local/bin bash install.sh" >&2
    rm -f "$tmp"
    exit 1
  fi

  echo "Installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
}

ensure_in_path() {
  case ":${PATH}:" in
    *:"${INSTALL_DIR}":*) return ;;
  esac

  echo ""
  echo "WARNING: ${INSTALL_DIR} is not in your PATH."
  echo "Add it by running one of the following, then restart your shell:"
  echo ""

  local shell_name
  shell_name="$(basename "${SHELL:-/bin/sh}")"

  if [ "$GOOS" = "windows" ]; then
    echo "  In Git Bash, add to ~/.bashrc:"
    echo "    echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc"
    echo ""
    echo "  Or add ${INSTALL_DIR} to your Windows PATH via System Environment Variables."
  else
    case "$shell_name" in
      zsh)
        echo "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.zshrc"
        ;;
      bash)
        echo "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.bashrc"
        ;;
      fish)
        echo "  fish_add_path ${INSTALL_DIR}"
        ;;
      *)
        echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
        ;;
    esac
  fi
  echo ""
}

detect_platform
INSTALL_DIR="${INSTALL_DIR:-${DEFAULT_INSTALL_DIR}}"
resolve_version
download_and_install
ensure_in_path
