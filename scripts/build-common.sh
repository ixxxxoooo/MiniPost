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
