# NetConnect

NetConnect is the native Linux desktop client for [NetLink](https://github.com/leonst036/NetLink). It creates a TUN interface to connect your machine directly to your NetLink overlay network without requiring a browser session.

## Requirements

- **Linux** with TUN kernel module (`/dev/net/tun`)
- **`iproute2`** (for IP routing configuration)
- **`libnotify`** (for desktop notifications)
- **Go 1.22+** and **Node.js 18+** (for building from source)
- **Wails v2** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

## Installation

Run the installer script with root privileges:

```bash
sudo ./install.sh
```

This compiles the binary, installs it to `/usr/local/bin/netconnect`, adds the `.desktop` launcher and application icons, and enables the systemd background daemon (`netconnect.service`).

To uninstall:

```bash
sudo ./uninstall.sh
```

## Development

Start the background daemon with root privileges:

```bash
sudo go run main.go
```

In another terminal, launch the GUI in development mode:

```bash
cd gui
wails dev -tags webkit2_41
```

## Usage

1. Open NetConnect.
2. Click the gear icon in the top-left to open **Settings**.
3. Click **Login with NetLink** to begin device authorization.
4. Open the verification URL in your browser, enter your NetLink password, and approve the connection.
5. Return to the main window and click the power button to connect.

## Configuration

Settings can be changed in the GUI or via environment variables:

| Variable | Default | Description |
|---|---|---|
| `NETLINK_RELAY_URL` | `http://localhost:4535` | URL of the NetLink relay server |
| `NETLINK_TARGET_ID` | `""` | Optional target device identifier |