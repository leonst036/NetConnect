#!/usr/bin/env bash
set -e

# Check root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "❌ Error: Installation requires root privileges."
    echo "👉 Please run: sudo ./install.sh (or curl -fsSL ... | sudo bash)"
    exit 1
fi

REPO="leonst036/NetConnect"
RELEASE_URL="https://github.com/${REPO}/releases/latest/download/netconnect-linux-amd64.tar.gz"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd || echo "")"
TEMP_DIR=""

cleanup() {
    if [ -n "${TEMP_DIR}" ] && [ -d "${TEMP_DIR}" ]; then
        rm -rf "${TEMP_DIR}"
    fi
}
trap cleanup EXIT

echo "========================================="
echo " Installing NetConnect"
echo "========================================="

# 1. Stop existing service if running
if systemctl is-active --quiet netconnect.service 2>/dev/null; then
    echo "[1/5] Stopping active netconnect service..."
    systemctl stop netconnect.service || true
fi

# 2. Acquire binary and assets (from local repo or latest GitHub release)
echo "[2/5] Preparing NetConnect application..."
BIN_SRC=""
ICON_PNG=""
ICON_SVG=""
DESKTOP_SRC=""
SERVICE_SRC=""
UNINSTALL_SRC=""

# Check if running inside local source repository with build capability
if [ -f "${SCRIPT_DIR}/gui/main.go" ]; then
    if [ -n "${SUDO_USER}" ]; then
        USER_HOME=$(eval echo "~${SUDO_USER}")
        sudo -u "${SUDO_USER}" bash -c "export PATH=\"${USER_HOME}/go/bin:${USER_HOME}/.local/bin:\$PATH\"; cd '${SCRIPT_DIR}/gui' && (wails build -tags webkit2_41 -o netconnect || true)"
    else
        export PATH="${HOME}/go/bin:${HOME}/.local/bin:${PATH}"
        (cd "${SCRIPT_DIR}/gui" && (wails build -tags webkit2_41 -o netconnect || true))
    fi

    if [ -f "${SCRIPT_DIR}/gui/build/bin/netconnect" ]; then
        BIN_SRC="${SCRIPT_DIR}/gui/build/bin/netconnect"
        ICON_PNG="${SCRIPT_DIR}/gui/build/appicon.png"
        ICON_SVG="${SCRIPT_DIR}/gui/frontend/src/assets/logo.svg"
        DESKTOP_SRC="${SCRIPT_DIR}/scripts/netconnect.desktop"
        SERVICE_SRC="${SCRIPT_DIR}/scripts/netconnect.service"
        UNINSTALL_SRC="${SCRIPT_DIR}/uninstall.sh"
    fi
fi

# Fallback or Remote: Download pre-built release package from GitHub
if [ -z "${BIN_SRC}" ] || [ ! -f "${BIN_SRC}" ]; then
    TEMP_DIR=$(mktemp -d)
    echo "Downloading latest release from GitHub (${REPO})..."
    if curl -sL --fail -o "${TEMP_DIR}/netconnect.tar.gz" "${RELEASE_URL}"; then
        tar -xzf "${TEMP_DIR}/netconnect.tar.gz" -C "${TEMP_DIR}"
        BIN_SRC="${TEMP_DIR}/netconnect"
        ICON_PNG="${TEMP_DIR}/netconnect.png"
        ICON_SVG="${TEMP_DIR}/logo.svg"
        DESKTOP_SRC="${TEMP_DIR}/netconnect.desktop"
        SERVICE_SRC="${TEMP_DIR}/netconnect.service"
        UNINSTALL_SRC="${TEMP_DIR}/uninstall.sh"
    else
        echo "❌ Error: Could not download release archive from ${RELEASE_URL}."
        echo "Please verify internet connection or build from source."
        exit 1
    fi
fi

# 3. Install Binary
echo "[3/5] Installing binary to /usr/local/bin/netconnect..."
install -m 755 "${BIN_SRC}" /usr/local/bin/netconnect

if command -v setcap >/dev/null 2>&1; then
    setcap cap_net_admin=ep /usr/local/bin/netconnect 2>/dev/null || true
fi

# 4. Install Desktop Launcher & Icons
echo "[4/5] Installing desktop launcher & application icons..."
mkdir -p /usr/share/icons/hicolor/512x512/apps
mkdir -p /usr/share/icons/hicolor/scalable/apps
[ -f "${ICON_PNG}" ] && cp "${ICON_PNG}" /usr/share/icons/hicolor/512x512/apps/netconnect.png
[ -f "${ICON_SVG}" ] && cp "${ICON_SVG}" /usr/share/icons/hicolor/scalable/apps/netconnect.svg

mkdir -p /usr/share/applications
[ -f "${DESKTOP_SRC}" ] && cp "${DESKTOP_SRC}" /usr/share/applications/netconnect.desktop
chmod 644 /usr/share/applications/netconnect.desktop 2>/dev/null || true

# Save uninstaller helper to system directory
mkdir -p /usr/local/share/netconnect
if [ -f "${UNINSTALL_SRC}" ]; then
    install -m 755 "${UNINSTALL_SRC}" /usr/local/share/netconnect/uninstall.sh
    ln -sf /usr/local/share/netconnect/uninstall.sh /usr/local/bin/netconnect-uninstall
fi

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

# 5. Setup & Start Systemd Background Service
echo "[5/5] Configuring systemd background service..."
if [ -f "${SERVICE_SRC}" ]; then
    cp "${SERVICE_SRC}" /etc/systemd/system/netconnect.service
    systemctl daemon-reload
    systemctl enable netconnect.service
    systemctl restart netconnect.service
fi

echo ""
echo "========================================="
echo " ✨ NetConnect installed successfully!"
echo "========================================="
echo " • Background Service: systemctl status netconnect"
echo " • Launch GUI:         netconnect (or from Application Menu)"
echo " • Uninstall:          sudo netconnect-uninstall"
echo "========================================="
