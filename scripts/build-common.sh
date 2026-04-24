#!/usr/bin/env bash

default_build_version() {
  git describe --tags --always --dirty 2>/dev/null || date +%Y.%m.%d.%H%M
}

sanitize_artifact_part() {
  local value="${1:-}"
  value="$(printf '%s' "$value" | tr -cs 'A-Za-z0-9._-' '-' | sed -E 's/-+/-/g; s/^-//; s/-$//')"

  if [[ -z "$value" ]]; then
    value="build"
  fi

  printf '%s' "$value"
}

normalize_arch_label() {
  case "$1" in
    amd64|x86_64) printf 'x64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) sanitize_artifact_part "$1" ;;
  esac
}

build_artifact_prefix() {
  local app_name version platform arch
  app_name="$(sanitize_artifact_part "$1")"
  version="$(sanitize_artifact_part "$2")"
  platform="$(sanitize_artifact_part "$3")"
  arch="$(normalize_arch_label "$4")"

  printf '%s-%s-%s-%s' "$app_name" "$version" "$platform" "$arch"
}

go_version_at_least() {
  local current="$1"
  local required="$2"
  local c_major c_minor c_patch r_major r_minor r_patch

  current="${current#go}"
  required="${required#go}"
  IFS=. read -r c_major c_minor c_patch <<<"$current"
  IFS=. read -r r_major r_minor r_patch <<<"$required"
  c_patch="${c_patch:-0}"
  r_patch="${r_patch:-0}"

  [[ "$c_major" =~ ^[0-9]+$ && "$c_minor" =~ ^[0-9]+$ && "$c_patch" =~ ^[0-9]+$ ]] || return 1
  [[ "$r_major" =~ ^[0-9]+$ && "$r_minor" =~ ^[0-9]+$ && "$r_patch" =~ ^[0-9]+$ ]] || return 1

  if (( c_major != r_major )); then
    (( c_major > r_major ))
    return
  fi
  if (( c_minor != r_minor )); then
    (( c_minor > r_minor ))
    return
  fi
  (( c_patch >= r_patch ))
}

require_patched_go_toolchain() {
  local required="${MINIPOST_MIN_GO_TOOLCHAIN:-go1.26.2}"
  local current

  if ! command -v go >/dev/null 2>&1; then
    echo "Missing command: go" >&2
    exit 1
  fi

  current="$(go env GOVERSION 2>/dev/null || true)"
  if [[ -z "$current" ]]; then
    current="$(go version | awk '{print $3}')"
  fi

  if ! go_version_at_least "$current" "$required"; then
    echo "Go toolchain $required or newer is required for MiniPost builds." >&2
    echo "Current toolchain: ${current:-unknown}" >&2
    echo "Run with Go's auto toolchain enabled or install $required+." >&2
    exit 1
  fi
}

clean_output_dir() {
  local out_dir="$1"
  local root_dir="$2"
  local parent abs_out abs_root

  parent="$(dirname "$out_dir")"
  mkdir -p "$parent"

  abs_out="$(cd "$parent" && pwd -P)/$(basename "$out_dir")"
  abs_root="$(cd "$root_dir" && pwd -P)"

  case "$abs_out" in
    /|"$HOME"|"$abs_root"|"$abs_root/build"|"$abs_root/build/bin")
      echo "Refusing to clean unsafe output directory: $abs_out" >&2
      exit 1
      ;;
  esac

  rm -rf "$abs_out"
  mkdir -p "$abs_out"
}
