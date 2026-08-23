#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_SRC="${SCRIPT_DIR}/gui/build/bin/netconnect"
ICON_PNG="${SCRIPT_DIR}/gui/build/appicon.png"
ICON_SVG="${SCRIPT_DIR}/gui/frontend/src/assets/logo.svg"

# Check root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "❌ Error: Installation requires root privileges."
    echo "👉 Please run: sudo ./install.sh"
    exit 1
fi

echo "========================================="
echo " Installing NetConnect on System"
echo "========================================="

# 1. Stop existing service if running to prevent file locks
if systemctl is-active --quiet netconnect.service 2>/dev/null; then
    echo "[1/5] Stopping active netconnect service..."
    systemctl stop netconnect.service || true
fi

# 2. Build NetConnect binary if missing or if rebuild requested
echo "[2/5] Building NetConnect application..."
if [ -n "${SUDO_USER}" ]; then
    sudo -u "${SUDO_USER}" bash -c "cd '${SCRIPT_DIR}/gui' && wails build -tags webkit2_41 -o netconnect"
else
    (cd "${SCRIPT_DIR}/gui" && wails build -tags webkit2_41 -o netconnect)
fi

# 3. Install Binary
echo "[3/5] Installing binary to /usr/local/bin/netconnect..."
install -m 755 "${BIN_SRC}" /usr/local/bin/netconnect

if command -v setcap >/dev/null 2>&1; then
    setcap cap_net_admin=ep /usr/local/bin/netconnect 2>/dev/null || true
fi

# 4. Install Desktop Entry
echo "[4/5] Installing desktop launcher & application icons..."
mkdir -p /usr/share/icons/hicolor/512x512/apps
mkdir -p /usr/share/icons/hicolor/scalable/apps
cp "${ICON_PNG}" /usr/share/icons/hicolor/512x512/apps/netconnect.png
cp "${ICON_SVG}" /usr/share/icons/hicolor/scalable/apps/netconnect.svg

mkdir -p /usr/share/applications
cp "${SCRIPT_DIR}/scripts/netconnect.desktop" /usr/share/applications/netconnect.desktop
chmod 644 /usr/share/applications/netconnect.desktop

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

# 5. Setup & Start Systemd Background Daemon
echo "[5/5] Configuring systemd background service..."
cp "${SCRIPT_DIR}/scripts/netconnect.service" /etc/systemd/system/netconnect.service
systemctl daemon-reload
systemctl enable netconnect.service
systemctl restart netconnect.service


echo ""
echo "========================================="
echo " ✨ NetConnect installed successfully!"
echo "========================================="
echo " • Background Service: systemctl status netconnect"
echo " • Launch GUI:         netconnect (or from Application Menu)"
echo " • Uninstall:          sudo ./uninstall.sh"
echo "========================================="
