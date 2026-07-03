#!/bin/sh
# Installer for dwarpal — https://github.com/YellowFoxH4XOR/dwarpal
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/YellowFoxH4XOR/dwarpal/main/install.sh | sh
#
# Env vars:
#   DWARPAL_VERSION   Version to install (default: latest release)
#   DWARPAL_INSTALL_DIR  Directory to install into (default: /usr/local/bin, falls back to ~/.local/bin)

set -e

REPO="YellowFoxH4XOR/dwarpal"
BIN_NAME="dwarpal"

log() { printf '%s\n' "$*" >&2; }
fail() { log "error: $*"; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command '$1' not found"
}

need_cmd curl
need_cmd tar
need_cmd uname

detect_os() {
  os=$(uname -s)
  case "$os" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) fail "unsupported OS: $os" ;;
  esac
}

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) fail "unsupported architecture: $arch" ;;
  esac
}

OS=$(detect_os)
ARCH=$(detect_arch)

if [ -z "${DWARPAL_VERSION:-}" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name":' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  [ -n "$VERSION" ] || fail "could not determine latest release version"
else
  VERSION="$DWARPAL_VERSION"
fi

VERSION_NO_V=${VERSION#v}

if [ "$OS" = "windows" ]; then
  EXT="zip"
else
  EXT="tar.gz"
fi

ARCHIVE="${BIN_NAME}_${VERSION_NO_V}_${OS}_${ARCH}.${EXT}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

log "Downloading ${URL}"
curl -fsSL "$URL" -o "${WORKDIR}/${ARCHIVE}" || fail "download failed: ${URL}"

log "Extracting ${ARCHIVE}"
if [ "$EXT" = "zip" ]; then
  need_cmd unzip
  unzip -q "${WORKDIR}/${ARCHIVE}" -d "$WORKDIR"
else
  tar -xzf "${WORKDIR}/${ARCHIVE}" -C "$WORKDIR"
fi

SRC_BIN="${WORKDIR}/${BIN_NAME}"
[ "$OS" = "windows" ] && SRC_BIN="${SRC_BIN}.exe"
[ -f "$SRC_BIN" ] || fail "binary not found in archive: ${SRC_BIN}"
chmod +x "$SRC_BIN"

# Pick install directory.
if [ -n "${DWARPAL_INSTALL_DIR:-}" ]; then
  INSTALL_DIR="$DWARPAL_INSTALL_DIR"
elif [ -w "/usr/local/bin" ] || [ "$(id -u)" = "0" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="${HOME}/.local/bin"
fi

mkdir -p "$INSTALL_DIR" 2>/dev/null || fail "could not create install directory: ${INSTALL_DIR}"
if [ ! -w "$INSTALL_DIR" ]; then
  fail "install directory is not writable: ${INSTALL_DIR}"
fi

DEST="${INSTALL_DIR}/${BIN_NAME}"
cp "$SRC_BIN" "$DEST"
chmod +x "$DEST"

# macOS Gatekeeper: curl-downloaded files carry com.apple.quarantine, and an
# unsigned quarantined binary is SIGKILLed (and removed) on first execution.
# Strip the attribute BEFORE the first run. Harmless no-op if absent.
if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "$DEST" 2>/dev/null || true
fi

# Verify the binary runs.
if ! "$DEST" version >/dev/null 2>&1; then
  fail "installed binary failed to run: ${DEST}"
fi

log "Installed ${BIN_NAME} ${VERSION} to ${DEST}"

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    log ""
    log "NOTE: ${INSTALL_DIR} is not on your PATH."
    log "  Add this to your shell profile:"
    log "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

log ""
log "Next steps:"
log "  ${BIN_NAME} version"
log "  ${BIN_NAME} init"
