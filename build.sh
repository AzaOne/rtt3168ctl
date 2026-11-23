#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$0")"

unformatted=$(gofmt -l cmd internal)
if [[ -n $unformatted ]]; then
    printf 'Run gofmt on these files:\n%s\n' "$unformatted" >&2
    exit 1
fi

go mod verify
export CGO_ENABLED=1
go vet ./...
(
    # The race detector runs on the build host, including for the linux/386 release.
    if [[ $(go env GOARCH) != "$(go env GOHOSTARCH)" || $(go env GOOS) != "$(go env GOHOSTOS)" ]]; then
        export GOARCH="$(go env GOHOSTARCH)" GOOS="$(go env GOHOSTOS)"
        unset CC CXX PKG_CONFIG_PATH PKG_CONFIG_LIBDIR
    fi
    go test -race ./...
)

mkdir -p build
version=$(git describe --tags --match '[0-9]*.[0-9]*.[0-9]*' --exclude '*[!0-9.]*' --exclude '*.*.*.*' --long --always --dirty 2>/dev/null || echo dev)
version_flag="-X rtt3168ctl/internal/interfaces/cli.buildVersion=$version"
# Keep linker options supplied by the caller (including release stripping).
build_args=()
while (( $# )); do
    case "$1" in
        -ldflags) build_args+=("-ldflags" "${2:?missing linker flags} $version_flag"); shift ;;
        -ldflags=*) build_args+=("$1 $version_flag") ;;
        *) build_args+=("$1") ;;
    esac
    shift
done
go build -trimpath -ldflags "$version_flag" "${build_args[@]}" -o build/rtt3168ctl ./cmd/rtt3168ctl
