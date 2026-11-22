#!/usr/bin/env bash
set -euo pipefail

tool="${1:?Informe a ferramenta a instalar.}"
destination="${2:-${RUNNER_TEMP:-/tmp}/ci-tools/bin}"
if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
  printf 'A instalação da CI exige Linux x86_64.\n' >&2
  exit 1
fi

case "$tool" in
  actionlint)
    repository=rhysd/actionlint
    version=1.7.12
    archive="actionlint_${version}_linux_amd64.tar.gz"
    expected_sha=8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8
    ;;
  gitleaks)
    repository=gitleaks/gitleaks
    version=8.30.1
    archive="gitleaks_${version}_linux_x64.tar.gz"
    expected_sha=551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb
    ;;
  trivy)
    repository=aquasecurity/trivy
    version=0.76.2
    archive="trivy_${version}_Linux-64bit.tar.gz"
    expected_sha=bc0fc726e017b8927a62b2d3531d7a84ee01769daf5c99ed866dced54089fdbf
    ;;
  *)
    printf 'Ferramenta não reconhecida: %s\n' "$tool" >&2
    exit 1
    ;;
esac

temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT
curl --fail --silent --show-error --location --retry 3 \
  "https://github.com/${repository}/releases/download/v${version}/${archive}" \
  --output "$temporary_directory/$archive"
printf '%s  %s\n' "$expected_sha" "$temporary_directory/$archive" | sha256sum --check --strict -
tar -xzf "$temporary_directory/$archive" -C "$temporary_directory" "$tool"
mkdir -p "$destination"
install -m 0755 "$temporary_directory/$tool" "$destination/$tool"
if [[ -n "${GITHUB_PATH:-}" ]]; then
  printf '%s\n' "$destination" >> "$GITHUB_PATH"
fi
printf '%s %s instalado com checksum verificado.\n' "$tool" "$version"
