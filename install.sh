#!/bin/sh
set -e

REPO="ofthemachine/meme-cli"
BINARY="meme-cli"
INSTALL_DIR="${MEME_CLI_INSTALL_DIR:-$HOME/.local/bin}"

# --- Colors (actual escape chars, safe in %s and format strings) ---
if [ -t 1 ]; then
  ESC=$(printf '\033')
  BOLD="${ESC}[1m"
  DIM="${ESC}[2m"
  GREEN="${ESC}[0;32m"
  CYAN="${ESC}[0;36m"
  YELLOW="${ESC}[0;33m"
  RED="${ESC}[0;31m"
  RESET="${ESC}[0m"
else
  BOLD='' DIM='' GREEN='' CYAN='' YELLOW='' RED='' RESET=''
fi

info()  { printf "%s=>%s %s\n" "$CYAN" "$RESET" "$1"; }
ok()    { printf "%s=>%s %s\n" "$GREEN" "$RESET" "$1"; }
warn()  { printf "%swarn:%s %s\n" "$YELLOW" "$RESET" "$1"; }
fail()  { printf "%serror:%s %s\n" "$RED" "$RESET" "$1" >&2; exit 1; }

# --- Banner ---
printf "\n"
printf "  %smeme-cli%s %s— generate memes from the command line%s\n" "$BOLD" "$RESET" "$DIM" "$RESET"
printf "  %shttps://github.com/%s%s\n" "$DIM" "$REPO" "$RESET"
printf "\n"

# --- Platform detection ---
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) fail "Unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    *) fail "Unsupported architecture: $(uname -m)" ;;
  esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"
PLATFORM="${OS}-${ARCH}"

info "Detected platform: ${BOLD}${PLATFORM}${RESET}"

# --- Find latest meme-cli release (tags are plain vX.Y.Z) ---
info "Finding latest release..."

RELEASES_URL="https://api.github.com/repos/${REPO}/releases"

if command -v curl >/dev/null 2>&1; then
  FETCH="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
  FETCH="wget -qO-"
else
  fail "Neither curl nor wget found. Install one and try again."
fi

RELEASES_JSON=$($FETCH "$RELEASES_URL") || fail "Could not fetch releases from GitHub."

TAG=$(printf '%s' "$RELEASES_JSON" | grep '"tag_name"' | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' | grep '^v[0-9]' | head -1)
[ -z "$TAG" ] && fail "No meme-cli release found. Check https://github.com/${REPO}/releases"

VERSION="$TAG"
info "Latest version: ${BOLD}${VERSION}${RESET}"

# --- Build asset URL ---
ASSET_NAME="${BINARY}-${PLATFORM}"
[ "$OS" = "windows" ] && ASSET_NAME="${ASSET_NAME}.exe"

DOWNLOAD_BASE="https://github.com/${REPO}/releases/download/${TAG}"
ASSET_URL="${DOWNLOAD_BASE}/${ASSET_NAME}"
CHECKSUMS_URL="${DOWNLOAD_BASE}/checksums.txt"

# --- Download binary ---
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${BOLD}${ASSET_NAME}${RESET}..."

if command -v curl >/dev/null 2>&1; then
  curl -fsSL -o "${TMPDIR}/${ASSET_NAME}" "$ASSET_URL" || fail "Download failed: ${ASSET_URL}"
  curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" 2>/dev/null || true
else
  wget -qO "${TMPDIR}/${ASSET_NAME}" "$ASSET_URL" || fail "Download failed: ${ASSET_URL}"
  wget -qO "${TMPDIR}/checksums.txt" "$CHECKSUMS_URL" 2>/dev/null || true
fi

# --- Verify checksum ---
if [ -s "${TMPDIR}/checksums.txt" ]; then
  EXPECTED=$(grep "${ASSET_NAME}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL=$(sha256sum "${TMPDIR}/${ASSET_NAME}" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL=$(shasum -a 256 "${TMPDIR}/${ASSET_NAME}" | awk '{print $1}')
    else
      warn "No sha256sum or shasum found — skipping checksum verification"
      ACTUAL="$EXPECTED"
    fi
    if [ "$EXPECTED" != "$ACTUAL" ]; then
      fail "Checksum mismatch! Expected: ${EXPECTED} Actual: ${ACTUAL}"
    fi
    ok "Checksum verified"
  fi
else
  warn "Could not download checksums — skipping verification"
fi

# --- Install ---
mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/${ASSET_NAME}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"
ok "Installed ${BOLD}${BINARY}${RESET} to ${INSTALL_DIR}/${BINARY}"

# --- PATH check ---
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    warn "${INSTALL_DIR} is not in your PATH"
    printf "\n"
    printf "  Add it to your shell profile:\n"
    printf "\n"
    printf "    %sexport PATH=\"%s:\$PATH\"%s\n" "$BOLD" "$INSTALL_DIR" "$RESET"
    printf "\n"
    ;;
esac

# --- Success ---
printf "\n"
printf "  %s%sReady to go!%s\n" "$GREEN" "$BOLD" "$RESET"
printf "\n"
printf "  %sMake a meme:%s\n" "$DIM" "$RESET"
printf "    %smeme-cli render drake \"writing your own installer\" \"curl | sh\"%s\n" "$BOLD" "$RESET"
printf "\n"
printf "  %sBrowse templates:%s\n" "$DIM" "$RESET"
printf "    %smeme-cli search <keyword>%s\n" "$BOLD" "$RESET"
printf "\n"
