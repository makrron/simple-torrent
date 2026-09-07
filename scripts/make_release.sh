#!/usr/bin/env bash
set -e

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
__root="$(cd "${__dir}/.." && pwd)"

GITVER=$(git describe --tags 2>/dev/null || echo "${GITHUB_REF_NAME:-v1.4.0}")
GO_LDFLAGS="-s -w -X main.VERSION=${GITVER}"
DIST_DIR="${DIST_DIR:-${__root}/dist}"

build_target() {
    local target_os="$1"
    local target_arch="$2"
    local target_arm="$3"

    local arch_label="${target_arch}"
    if [[ -n "${target_arm}" ]]; then
        arch_label="armv${target_arm}"
    fi

    echo "==> Building ${target_os}/${arch_label} (${GITVER})..."
    local temp_bin_dir="${DIST_DIR}/bin_${target_os}_${arch_label}"
    mkdir -p "${temp_bin_dir}"

    local bin_name="cloud-torrent"
    local alt_name="simple-torrent"
    if [[ "${target_os}" == "windows" ]]; then
        bin_name="cloud-torrent.exe"
        alt_name="simple-torrent.exe"
    fi

    (
        cd "${__root}"
        CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" GOARM="${target_arm}" \
            go build -trimpath -ldflags "${GO_LDFLAGS}" -o "${temp_bin_dir}/${bin_name}" .
    )

    cp "${temp_bin_dir}/${bin_name}" "${temp_bin_dir}/${alt_name}"

    # Legacy standalone .gz binaries for direct curl installation
    if [[ "${target_os}" != "windows" ]]; then
        cp "${temp_bin_dir}/${bin_name}" "${DIST_DIR}/cloud-torrent_${target_os}_${arch_label}_static"
        gzip -f -9 "${DIST_DIR}/cloud-torrent_${target_os}_${arch_label}_static"

        cp "${temp_bin_dir}/${alt_name}" "${DIST_DIR}/simple-torrent_${target_os}_${arch_label}_static"
        gzip -f -9 "${DIST_DIR}/simple-torrent_${target_os}_${arch_label}_static"
    fi

    # Standard distribution archive (.zip for windows, .tar.gz for unix)
    if [[ "${target_os}" == "windows" ]]; then
        if command -v zip >/dev/null 2>&1; then
            (cd "${temp_bin_dir}" && zip -q -9 "${DIST_DIR}/simple-torrent_${target_os}_${arch_label}.zip" "${bin_name}" "${alt_name}")
        else
            tar -czf "${DIST_DIR}/simple-torrent_${target_os}_${arch_label}.tar.gz" -C "${temp_bin_dir}" "${bin_name}" "${alt_name}"
        fi
    else
        tar -czf "${DIST_DIR}/simple-torrent_${target_os}_${arch_label}.tar.gz" -C "${temp_bin_dir}" "${bin_name}" "${alt_name}"
    fi

    rm -rf "${temp_bin_dir}"
}

build_all() {
    mkdir -p "${DIST_DIR}"
    echo "Starting full multi-platform release build for ${GITVER}..."

    # Linux platforms
    build_target "linux" "amd64" ""
    build_target "linux" "arm64" ""
    build_target "linux" "arm" "7"
    build_target "linux" "386" ""

    # macOS / Darwin
    build_target "darwin" "amd64" ""
    build_target "darwin" "arm64" ""

    # Windows (x64)
    build_target "windows" "amd64" ""

    echo "==> Generating SHA256 checksums..."
    (
        cd "${DIST_DIR}"
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum *.{tar.gz,zip,gz} 2>/dev/null > checksums.txt || true
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 *.{tar.gz,zip,gz} 2>/dev/null > checksums.txt || true
        fi
    )
    echo "==> Build complete! Output artifacts in: ${DIST_DIR}"
    ls -lh "${DIST_DIR}"
}

# Legacy single-target compatibility mode
if [[ "$1" == "all" || "$1" == "--all" || -z "$1" ]]; then
    build_all
else
    BINPREFIX=""
    BIN=cloud-torrent
    if [[ -d ${BINLOCATION} ]]; then
        BINPREFIX=${BINLOCATION}/
    fi

    OS=""
    ARCH=""
    SUFFIX=""
    OSSUFFIX=""
    PKGCMD=
    CGO=1
    GO_TAGS=""

    for arg in "$@"; do
        case $arg in
            amd64) ARCH=amd64 ;;
            arm64) ARCH=arm64 ;;
            386) ARCH=386 ;;
            arm) ARCH=arm ;;
            windows) OS=windows; OSSUFFIX=.exe ;;
            darwin) OS=darwin ;;
            xz) PKGCMD=xz ;;
            gzip) PKGCMD=gzip ;;
            purego) CGO=0; SUFFIX=_static ;;
            static) CGO=1; SUFFIX=_static; GO_LDFLAGS="${GO_LDFLAGS} -extldflags=-static"; GO_TAGS='netgo osusergo sqlite_omit_load_extension' ;;
        esac
    done

    if [[ -z $OS ]]; then OS=$(go env GOOS); fi
    if [[ -z $ARCH ]]; then ARCH=$(go env GOARCH); fi

    BINFILE=${BINPREFIX}${BIN}_${OS}_${ARCH}${SUFFIX}${OSSUFFIX}
    CGO_ENABLED=$CGO GOARCH=$ARCH GOOS=$OS \
        go build -o "${BINFILE}" -trimpath -ldflags "${GO_LDFLAGS}" -tags "${GO_TAGS}"
    
    if [[ ! -f ${BINFILE} ]]; then
        echo "Build failed."
        exit 1
    fi

    if [[ -n $PKGCMD ]]; then
        ${PKGCMD} -v -9 "${BINFILE}"
    fi
fi
