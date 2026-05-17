#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="mediakit-cli"
RELEASE_BASE_URL="${MEDIKIT_CLI_RELEASE_BASE_URL:-https://github.com/volcengine/mediakit-cli/releases/download}"
LATEST_API_URL="${MEDIKIT_CLI_LATEST_API_URL:-https://api.github.com/repos/volcengine/mediakit-cli/releases/latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-}"

log() {
  printf '[%s] %s\n' "${PROJECT_NAME}" "$1"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

resolve_os() {
  case "$(uname -s)" in
    Darwin) echo "darwin" ;;
    Linux) echo "linux" ;;
    *)
      echo "unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [[ -n "${VERSION}" ]]; then
    echo "${VERSION#v}"
    return
  fi

  require_cmd curl
  local response
  response="$(curl -fsSL "${LATEST_API_URL}")"
  local tag
  tag="$(printf '%s' "${response}" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name":[[:space:]]*"([^"]+)".*/\1/')"
  if [[ -z "${tag}" ]]; then
    echo "failed to resolve latest release tag" >&2
    exit 1
  fi
  echo "${tag#v}"
}

sha256_check() {
  local archive_path="$1"
  local checksums_path="$2"
  local archive_name
  archive_name="$(basename "${archive_path}")"
  local expected
  expected="$(grep " ${archive_name}$" "${checksums_path}" | awk '{print $1}')"
  if [[ -z "${expected}" ]]; then
    echo "checksum not found for ${archive_name}" >&2
    exit 1
  fi

  local actual=""
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${archive_path}" | awk '{print $1}')"
  else
    actual="$(shasum -a 256 "${archive_path}" | awk '{print $1}')"
  fi

  if [[ "${actual}" != "${expected}" ]]; then
    echo "checksum mismatch for ${archive_name}" >&2
    exit 1
  fi
}

main() {
  require_cmd curl
  require_cmd tar
  require_cmd install

  local os arch version archive ext tmp_dir archive_path checksums_path binary_path
  os="$(resolve_os)"
  arch="$(resolve_arch)"
  version="$(resolve_version)"
  ext="tar.gz"
  archive="${PROJECT_NAME}_${version}_${os}_${arch}.${ext}"

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT

  archive_path="${tmp_dir}/${archive}"
  checksums_path="${tmp_dir}/checksums.txt"

  log "downloading ${archive}"
  curl -fsSL "${RELEASE_BASE_URL}/v${version}/${archive}" -o "${archive_path}"
  curl -fsSL "${RELEASE_BASE_URL}/v${version}/checksums.txt" -o "${checksums_path}"

  sha256_check "${archive_path}" "${checksums_path}"

  tar -xzf "${archive_path}" -C "${tmp_dir}"
  binary_path="${tmp_dir}/${PROJECT_NAME}"
  if [[ ! -f "${binary_path}" ]]; then
    echo "binary not found after extract: ${binary_path}" >&2
    exit 1
  fi

  mkdir -p "${INSTALL_DIR}"
  install -m 755 "${binary_path}" "${INSTALL_DIR}/${PROJECT_NAME}"
  log "installed ${INSTALL_DIR}/${PROJECT_NAME}"
  log "run '${PROJECT_NAME} version' to verify the installation"
}

main "$@"
