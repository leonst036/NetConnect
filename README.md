# NetConnect

NetConnect is the desktop version (currently for Linux) of [NetLink](https://github.com/leonst036/NetLink). With this program, you can connect to your NetLink Network natively on your machine without using a browser and without a VPN.

> [!WARNING]
> **THIS PROJECT IS IN A VERY EARLY STAGE OF DEVELOPMENT!**

---

## Requirements

### Operating System
- **Linux** (with TUN/TAP kernel module support enabled at `/dev/net/tun`)

### System Packages & Tools
- **`iproute2`** (provides the `ip` command used to configure TUN interfaces and IP routes)
- **`libnotify` / `libnotify-bin`** (provides `notify-send` for desktop status notifications)
- **D-Bus notification daemon** (standard on most desktop environments like GNOME, KDE, XFCE)

### Go Environment
- **Go 1.22+** (configured with Go 1.26+ in `go.mod`)

### Permissions
NetConnect requires network administration privileges to create TUN interfaces and manipulate routing tables. You can run it either:
- With **`sudo` / root**:
  ```bash
  sudo go run .
  ```
- Or by granting Linux capabilities to the compiled binary:
  ```bash
  go build -o netconnect .
  sudo setcap cap_net_admin,cap_net_raw=eip ./netconnect
  ./netconnect
  ```

### You need to install the NetStore app net-graph for this to work

---

## Configuration

NetConnect can be configured via environment variables or directly inside the desktop GUI settings:

| Variable | Default | Description |
|---|---|---|
| `NETLINK_RELAY_URL` | `http://localhost:4535` | The URL of the NetLink relay server |