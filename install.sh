#!/bin/sh
# Installs schemalens, and optionally adapters, from a GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/mokevnin/schemalens/main/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- -b ~/.local/bin -a ent -a gorm v0.1.3
#
# Usage: install.sh [-b <dir>] [-a <adapter>]... [<version>]
#   -b <dir>      install directory (default: /usr/local/bin, or ~/.local/bin if that is not writable)
#   -a <adapter>  also install schemalens-adapter-<adapter> (ent, gorm or bun); repeatable
#   <version>     release tag such as v0.1.3 (default: latest)

set -eu

repo="mokevnin/schemalens"
bindir=""
adapters=""
version=""

usage() {
	cat >&2 <<'EOF_USAGE'
Usage: install.sh [-b <dir>] [-a <adapter>]... [<version>]
  -b <dir>      install directory (default: /usr/local/bin, or ~/.local/bin if that is not writable)
  -a <adapter>  also install schemalens-adapter-<adapter> (ent, gorm or bun); repeatable
  <version>     release tag such as v0.1.3 (default: latest)
EOF_USAGE
	exit 2
}

fail() {
	echo "install.sh: $*" >&2
	exit 1
}

while [ $# -gt 0 ]; do
	case "$1" in
	-b) [ $# -ge 2 ] || usage; bindir="$2"; shift 2 ;;
	-a) [ $# -ge 2 ] || usage; adapters="$adapters $2"; shift 2 ;;
	-h | --help) usage ;;
	-*) usage ;;
	*) version="$1"; shift ;;
	esac
done

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported OS $(uname -s); download a release archive from https://github.com/$repo/releases" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "unsupported architecture $(uname -m)" ;;
esac

for tool in curl tar; do
	command -v "$tool" >/dev/null 2>&1 || fail "$tool is required"
done

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
	fail "sha256sum or shasum is required"
fi

if [ -z "$version" ]; then
	# The redirect target of /releases/latest ends in the tag; this avoids the API rate limit.
	version=$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest") ||
		fail "cannot resolve the latest release"
	version=${version##*/}
fi
case "$version" in
v[0-9]*) ;;
[0-9]*) version="v$version" ;;
*) fail "invalid version $version" ;;
esac
number=${version#v}

if [ -z "$bindir" ]; then
	if [ -w /usr/local/bin ]; then
		bindir=/usr/local/bin
	else
		bindir="$HOME/.local/bin"
	fi
fi
mkdir -p "$bindir" || fail "cannot create $bindir"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

base="https://github.com/$repo/releases/download/$version"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "cannot download checksums for $version"

install_binary() {
	name="$1"
	archive="${name}_${number}_${os}_${arch}.tar.gz"
	curl -fsSL -o "$tmp/$archive" "$base/$archive" || fail "cannot download $archive"
	want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
	[ -n "$want" ] || fail "$archive is not listed in checksums.txt"
	[ "$(sha256 "$tmp/$archive")" = "$want" ] || fail "checksum mismatch for $archive"
	mkdir "$tmp/$name"
	tar -xzf "$tmp/$archive" -C "$tmp/$name" "$name"
	mv -f "$tmp/$name/$name" "$bindir/$name"
	chmod 755 "$bindir/$name"
	echo "installed $name $version to $bindir/$name"
}

install_binary schemalens
for adapter in $adapters; do
	case "$adapter" in
	ent | gorm | bun) install_binary "schemalens-adapter-$adapter" ;;
	*) fail "unknown adapter $adapter (want ent, gorm or bun)" ;;
	esac
done

case ":$PATH:" in
*":$bindir:"*) ;;
*) echo "note: $bindir is not on PATH" >&2 ;;
esac
