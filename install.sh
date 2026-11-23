#!/usr/bin/env bash
set -euo pipefail

usage() {
    echo 'Usage: ./install.sh [--user] [--udev]'
    echo 'Install the built CLI in ~/.local/bin. Add --udev to install the USB access rule with sudo.'
}

with_udev=false
for arg in "$@"; do
    case "$arg" in
        --user) ;;
        --udev) with_udev=true ;;
        -h|--help) usage; exit 0 ;;
        *) usage >&2; exit 2 ;;
    esac
done

if (( EUID == 0 )); then
    echo 'Run the installer as the user who will use the CLI.' >&2
    exit 1
fi
cd -- "$(dirname -- "$0")"
if [[ $(uname -s) != Linux ]]; then
    echo 'rtt3168ctl requires Linux.' >&2
    exit 1
fi
binary=build/rtt3168ctl
if [[ ! -f $binary ]]; then
    echo "Missing $binary. Run ./build.sh first." >&2
    exit 1
fi

bindir="$HOME/.local/bin"
mkdir -p -- "$bindir"
stage=$(mktemp -- "$bindir/.rtt3168ctl.XXXXXXXX")
trap 'rm -f -- "$stage"' EXIT
install -m 0755 -- "$binary" "$stage"
mv -fT -- "$stage" "$bindir/rtt3168ctl"
echo "Installed $bindir/rtt3168ctl"

if "$with_udev"; then
    rule=52-rtt3168ctl-093a-2533.rules
    sudo install -D -m 0644 -- "$rule" "/etc/udev/rules.d/$rule"
    sudo udevadm control --reload-rules
    echo 'Reconnect the mouse to apply the USB access rule.'
fi
