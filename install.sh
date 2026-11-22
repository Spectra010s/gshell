#!/usr/bin/env sh
#
# Install a prebuilt gshell release.
#
# Usage:
#   ./install.sh
#   GSHELL_VERSION=2.0.0 ./install.sh
#   GSHELL_INSTALL_DIR=~/bin ./install.sh
#   PREFIX=/usr/local ./install.sh

set -eu

REPO=Spectra010s/gshell

say() { printf '%s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }
# Termux has no /etc/resolv.conf, so a prebuilt Linux binary cannot resolve
# DNS there. Termux users get gshell from termux-packages instead.
if [ -n "${TERMUX_VERSION:-}" ] || [ "$(uname -o 2>/dev/null || true)" = "Android" ]; then
	die "Termux is not supported by this installer; install gshell from termux-packages instead"
fi

for cmd in curl tar; do
	command -v "$cmd" >/dev/null 2>&1 || die "$cmd is required"
done

case "$(uname -s)" in
Linux) os=Linux ;;
Darwin) os=Darwin ;;
*) die "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=x86_64 ;;
arm64 | aarch64) arch=arm64 ;;
*) die "unsupported architecture: $(uname -m)" ;;
esac

version=${GSHELL_VERSION:-}
if [ -z "$version" ]; then
	url=$(curl -fsSL -o /dev/null -w '%{url_effective}' \
		"https://github.com/$REPO/releases/latest") ||
		die "could not determine the latest release"
	version=${url##*/tag/}
	case $version in
	v[0-9]*) version=${version#v} ;;
	*) die "could not determine the latest release" ;;
	esac
fi
version=${version#v}

asset="gshell_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/v${version}"

tmp=$(mktemp -d "${TMPDIR:-/tmp}/gshell.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading $asset"
curl -fSL --progress-bar -o "$tmp/$asset" "$base/$asset" ||
	die "download failed; does v$version exist?"

say "Verifying checksum"
curl -fsSL -o "$tmp/checksums.txt" "$base/gshell_${version}_checksums.txt" ||
	die "could not download checksums"
expected=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$expected" ] || die "no checksum found for $asset"

if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$asset" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
	actual=$(shasum -a 256 "$tmp/$asset" | awk '{print $1}')
elif command -v openssl >/dev/null 2>&1; then
	actual=$(openssl dgst -sha256 "$tmp/$asset" | awk '{print $NF}')
else
	die "no sha256 tool found (sha256sum, shasum, or openssl)"
fi
[ "$actual" = "$expected" ] || die "checksum mismatch for $asset"

tar -xzf "$tmp/$asset" -C "$tmp"
[ -f "$tmp/gshell" ] || die "archive does not contain gshell"

if [ -n "${GSHELL_INSTALL_DIR:-}" ]; then
	dir=$GSHELL_INSTALL_DIR
elif [ -n "${PREFIX:-}" ]; then
	dir=$PREFIX/bin
else
	dir=$HOME/.local/bin
fi
case $dir in
/*) ;;
*) dir=$PWD/$dir ;;
esac

mkdir -p "$dir"
install -m 0755 "$tmp/gshell" "$dir/gshell"
printf 'Installed %s/gshell (%s)\n' "$dir" "$version"

case ":$PATH:" in
*":$dir:"*) ;;
*) say "Note: $dir is not in PATH." ;;
esac

say "For shell completions, run: gshell completion bash  (or zsh, fish)"
