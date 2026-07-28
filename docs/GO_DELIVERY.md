# Go Cross-Platform Delivery

This document is the delivery checklist for the Go gateway edition of Residential IP Manager.

## Scope

The Go edition provides:

- Cross-platform `rim` binary for Linux, Windows, and macOS.
- Local HTML control console with cookie login.
- Username and password loaded from the backend JSON config.
- VPNGate node refresh and safe OpenVPN profile parsing.
- Conservative residential-IP classification.
- TCP availability probing with bounded concurrency.
- In-process OpenVPN protocol handling with sanitized VPNGate profiles.
- Userspace TCP/IP networking with no system TUN device or elevated privilege.
- Public exit verification after connection.
- Automatic failover across candidate nodes with cooldown.
- Periodic end-to-end exit verification with a configurable consecutive-failure threshold.
- In-process VMESS inbound and loopback SOCKS proxy.
- VMESS and Clash subscription endpoints protected by token.
- Gateway automation that starts VMESS and connects the VPNGate/OpenVPN exit on service startup.
- JSON state persistence.
- Service templates for systemd, launchd, and Windows `sc.exe`.
- CI and tag release workflows for Go cross-platform artifacts.

## Runtime Topology

```text
local apps on deployed machine
  -> 127.0.0.1:1080 SOCKS
  -> embedded proxy core
  -> userspace TCP/IP stack
  -> built-in OpenVPN protocol engine
  -> VPNGate selected node
  -> Internet

VMESS client on localhost, LAN, or Internet
  -> embedded VMESS inbound on this host
  -> userspace TCP/IP stack
  -> built-in OpenVPN protocol engine
  -> VPNGate selected node
  -> Internet

```

The deployed machine is always the gateway. Local programs opt into its SOCKS listener; LAN or public clients use VMESS. The app does not replace the operating-system default route, which is what allows it to work without a TUN driver or administrator privileges.

## Required Components

- `rim` Go binary for the target OS.
- No OpenVPN, xray, Python, TUN driver, or other runtime installation.
- No administrator/root privilege for the userspace proxy data path.
- Outbound Internet access to the VPNGate API and selected VPNGate server.
- A reachable `subscription.host:subscription.port` for clients outside the machine.

## First Run

```bash
rim config init --output config.json
rim serve --config config.json
# or
rim serve --config config.json --listen 127.0.0.1:8899
```

Open:

```text
http://127.0.0.1:8899
```

`rim config init` generates and prints the Web password once. It also generates random values for `server.session_secret`, `subscription.token`, and `subscription.uuid`.

Review these before exposing the console or subscription outside localhost:

- `auth.password`
- `server.session_secret`
- `subscription.token`
- `subscription.host`
- `subscription.uuid`

Default gateway automation:

```json
"deployment": {
  "mode": "gateway",
  "auto_start_vmess": true,
  "auto_connect_exit": true
}
```

Set `subscription.host` to `auto` for same-machine or LAN use. In `auto` mode the app prefers a LAN/private address first, then falls back to a public address when needed. Set it to a public IP or domain when deploying a public gateway.

Gateway mode requires `subscription.enabled=true` and `proxy_core.enabled=true`; that is the public-shareable VMESS gateway behavior.

## Main API Checks

Login:

```bash
curl -c cookies.txt \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"<generated-password>"}' \
  http://127.0.0.1:8899/api/login
```

Environment:

```bash
curl -b cookies.txt http://127.0.0.1:8899/api/environment
curl -b cookies.txt http://127.0.0.1:8899/api/proxy/status
```

Node lifecycle:

```bash
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/nodes/refresh
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/nodes/classify
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/nodes/probe
curl -b cookies.txt http://127.0.0.1:8899/api/nodes
```

Connection:

```bash
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/connect
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/exit/verify
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/disconnect
```

VMESS core:

```bash
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/proxy/start
curl -b cookies.txt -X POST http://127.0.0.1:8899/api/proxy/stop
```

Subscriptions:

```bash
curl 'http://127.0.0.1:8898/sub/vmess?token=<subscription.token>'
curl 'http://127.0.0.1:8898/sub/clash?token=<subscription.token>'
```

## Service Templates

Generate service files:

```bash
rim service generate --platform linux --output service/linux \
  --binary /usr/local/bin/rim \
  --config /etc/residential-ip-manager/config.json \
  --data-dir /var/lib/residential-ip-manager

rim service generate --platform darwin --output service/darwin \
  --binary /usr/local/bin/rim \
  --config /usr/local/etc/residential-ip-manager/config.json \
  --data-dir /usr/local/var/residential-ip-manager

rim service generate --platform windows --output service/windows \
  --binary "C:\Program Files\ResidentialIPManager\rim.exe" \
  --config "C:\ProgramData\ResidentialIPManager\config.json"
```

Release builds also generate:

```text
dist/go/service/linux/residential-ip-manager.service
dist/go/service/darwin/com.guli-joy.residential-ip-manager.plist
dist/go/service/windows/install-service.ps1
dist/go/service/windows/uninstall-service.ps1
```

## Release Checklist

Run before tagging:

```bash
go test ./...
(cd third_party/minivpn && go test ./...)
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
(cd third_party/minivpn && go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...)
OUT_DIR=dist/go VERSION=<tag> sh scripts/build_go.sh
```

Verify:

- `dist/go/rim-linux-amd64`
- `dist/go/rim-linux-arm64`
- `dist/go/rim-windows-amd64.exe`
- `dist/go/rim-darwin-amd64`
- `dist/go/rim-darwin-arm64`
- `dist/go/packages/residential-ip-manager-<tag>-linux-amd64.tar.gz`
- `dist/go/packages/residential-ip-manager-<tag>-linux-arm64.tar.gz`
- `dist/go/packages/residential-ip-manager-<tag>-windows-amd64.zip`
- `dist/go/packages/residential-ip-manager-<tag>-darwin-amd64.tar.gz`
- `dist/go/packages/residential-ip-manager-<tag>-darwin-arm64.tar.gz`
- `dist/go/SHA256SUMS.txt`
- service templates under `dist/go/service/`

Each package contains:

- The target-platform `rim` binary.
- `configs/go.example.json`.
- `docs/GO_DELIVERY.md`.
- `README.md`, `SECURITY.md`, `LICENSE`, third-party notices, and license texts.
- The matching service template for that platform.

Runtime smoke test:

```bash
dist/go/rim-darwin-arm64 config init --output /tmp/rim-config.json
dist/go/rim-darwin-arm64 serve --config /tmp/rim-config.json --listen 127.0.0.1:8899
```

Then verify:

- `/` returns the login page.
- `/api/status` returns `401` before login.
- `/api/login` accepts the configured credentials.
- `/api/environment` reports the built-in OpenVPN tunnel and embedded proxy status.
- `/sub/vmess` returns `401` without token.
- `/sub/vmess?token=...` returns a base64 VMESS subscription.

## Security Notes

- Keep `server.listen` on `127.0.0.1` unless the host is protected by firewall and TLS/reverse proxy.
- Treat `subscription.token` like a password.
- Do not commit generated `config.json`, state files, or OpenVPN profiles.
- Review the linked OpenVPN and VMESS engine versions independently.
- Public VPNGate nodes are untrusted inputs.
- The project is distributed under GPL-3.0 because the linked minivpn implementation is GPL-3.0.
- Publish releases from signed repository tags so the GitHub source archive exactly matches each binary release.
