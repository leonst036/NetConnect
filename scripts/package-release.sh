#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
STAGE_DIR="${DIST_DIR}/release_stage"

echo "Building production NetConnect binary..."
cd "${ROOT_DIR}/gui"
wails build -tags webkit2_41 -o netconnect

echo "Creating release archive..."
rm -rf "${STAGE_DIR}" "${DIST_DIR}/netconnect-linux-amd64.tar.gz"
mkdir -p "${STAGE_DIR}"

cp "${ROOT_DIR}/gui/build/bin/netconnect" "${STAGE_DIR}/"
cp "${ROOT_DIR}/gui/build/appicon.png" "${STAGE_DIR}/netconnect.png"
cp "${ROOT_DIR}/gui/frontend/src/assets/logo.svg" "${STAGE_DIR}/logo.svg"
cp "${ROOT_DIR}/scripts/netconnect.desktop" "${STAGE_DIR}/"
cp "${ROOT_DIR}/scripts/netconnect.service" "${STAGE_DIR}/"
cp "${ROOT_DIR}/install.sh" "${STAGE_DIR}/"
cp "${ROOT_DIR}/uninstall.sh" "${STAGE_DIR}/"

cd "${STAGE_DIR}"
tar -czf "${DIST_DIR}/netconnect-linux-amd64.tar.gz" *

rm -rf "${STAGE_DIR}"
echo "Release archive created: ${DIST_DIR}/netconnect-linux-amd64.tar.gz"
