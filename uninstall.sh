#!/usr/bin/env bash
set -e

# Check root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "❌ Error: Uninstallation requires root privileges."
    echo "👉 Please run: sudo ./uninstall.sh"
    exit 1
fi

echo "========================================="
echo " Uninstalling NetConnect from System"
echo "========================================="

# 1. Stop all running NetConnect instances and services
echo "[1/4] Stopping all active NetConnect processes and services..."


# Stop systemd system service
if systemctl is-active --quiet netconnect.service 2>/dev/null; then
    systemctl stop netconnect.service || true
fi
if systemctl is-enabled --quiet netconnect.service 2>/dev/null; then
    systemctl disable netconnect.service || true
fi
rm -f /etc/systemd/system/netconnect.service
systemctl daemon-reload

# Stop user services if running under sudo
if [ -n "${SUDO_USER}" ]; then
    sudo -u "${SUDO_USER}" bash -c "systemctl --user stop netconnect.service 2>/dev/null || true; systemctl --user disable netconnect.service 2>/dev/null || true"
fi

# Kill any remaining running netconnect processes
killall netconnect 2>/dev/null || pkill -x netconnect 2>/dev/null || true


# 2. Remove Binary
echo "[2/4] Removing binary..."
rm -f /usr/local/bin/netconnect

# 3. Remove Desktop Launcher
echo "[3/4] Removing desktop launcher..."
rm -f /usr/share/applications/netconnect.desktop

# 4. Remove Icons
echo "[4/4] Removing application icons..."
rm -f /usr/share/icons/hicolor/512x512/apps/netconnect.png
rm -f /usr/share/icons/hicolor/scalable/apps/netconnect.svg

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor >/dev/null 2>&1 || true
fi

echo "========================================="
echo " ✨ NetConnect uninstalled successfully."
echo "========================================="
