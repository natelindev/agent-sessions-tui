#!/bin/sh

set -eu

program_name="agent-sessions-tui"
minimum_go_minor=24

usage() {
    cat <<'EOF'
Build and install agent-sessions-tui from this checkout.

Usage:
  ./scripts/install.sh [--bin-dir PATH]

Options:
  --bin-dir PATH  Installation directory (default: $XDG_BIN_HOME or ~/.local/bin)
  -h, --help      Show this help

Examples:
  ./scripts/install.sh
  ./scripts/install.sh --bin-dir /usr/local/bin
EOF
}

fail() {
    printf 'error: %s\n' "$1"
    printf 'help: Run ./scripts/install.sh --help for usage.\n'
    exit 2
}

if [ -n "${XDG_BIN_HOME:-}" ]; then
    bin_dir=$XDG_BIN_HOME
else
    [ -n "${HOME:-}" ] || fail 'HOME is not set and --bin-dir was not provided'
    bin_dir=$HOME/.local/bin
fi

while [ "$#" -gt 0 ]; do
    case "$1" in
        --bin-dir)
            [ "$#" -ge 2 ] || fail '--bin-dir requires a path'
            bin_dir=$2
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            fail "unknown option: $1"
            ;;
    esac
done

case "$bin_dir" in
    ""|/) fail 'installation directory must not be empty or /' ;;
esac

command -v go >/dev/null 2>&1 || fail 'Go 1.24 or newer is required'

go_version=$(go env GOVERSION 2>/dev/null || true)
go_version=${go_version#go}
go_major=${go_version%%.*}
go_remainder=${go_version#*.}
go_minor=${go_remainder%%.*}
case "$go_major:$go_minor" in
    *[!0-9:]*|:*) fail "cannot determine the installed Go version: $go_version" ;;
esac
if [ "$go_major" -lt 1 ] || { [ "$go_major" -eq 1 ] && [ "$go_minor" -lt "$minimum_go_minor" ]; }; then
    fail "Go 1.$minimum_go_minor or newer is required; found go$go_version"
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

mkdir -p "$bin_dir" || fail "cannot create installation directory: $bin_dir"
bin_dir=$(CDPATH= cd -- "$bin_dir" && pwd) || fail "cannot resolve installation directory: $bin_dir"
temp_dir=$(mktemp -d "$bin_dir/.agent-sessions-tui-install.XXXXXX") || fail "cannot create a temporary directory in: $bin_dir"
temp_binary=$temp_dir/$program_name
cleanup() {
    rm -f -- "$temp_binary"
    rmdir "$temp_dir" 2>/dev/null || true
}
trap cleanup EXIT HUP INT TERM

version=$(sed -n 's/^var version = "\([^"]*\)"$/\1/p' "$repo_root/cmd/agent-sessions-tui/main.go")
[ -n "$version" ] || version=dev

if ! (cd "$repo_root" && go build -trimpath -ldflags "-s -w -X main.version=$version" -o "$temp_binary" ./cmd/agent-sessions-tui); then
    fail 'build failed'
fi
chmod 755 "$temp_binary"

destination=$bin_dir/$program_name
status=installed
if [ -f "$destination" ] && cmp -s "$temp_binary" "$destination"; then
    status=current
else
    mv -f -- "$temp_binary" "$destination" || fail "cannot install binary at: $destination"
fi

display_destination=$destination
display_bin_dir=$bin_dir
if [ -n "${HOME:-}" ]; then
    case "$destination" in
        "$HOME"/*) display_destination="~/${destination#"$HOME"/}" ;;
    esac
    case "$bin_dir" in
        "$HOME"/*) display_bin_dir="~/${bin_dir#"$HOME"/}" ;;
    esac
fi

printf 'install:\n'
printf '  status: %s\n' "$status"
printf '  bin: %s\n' "$display_destination"
printf '  version: %s\n' "$version"

case :${PATH:-}: in
    *:"$bin_dir":*) ;;
    *)
        printf 'help[1]: Add %s to PATH, then run agent-sessions-tui\n' "$display_bin_dir"
        ;;
esac
