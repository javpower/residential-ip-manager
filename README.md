# Residential IP Manager Go

Cross-platform, single-binary VPNGate gateway with a local SOCKS proxy, VMESS service, subscriptions, and an authenticated Web console.

The deployed machine first connects to a VPNGate/OpenVPN exit. It then exposes a VMESS inbound so the same box can serve itself, a local LAN, or public subscribers through one shared gateway model.

## What It Does

- Runs on Linux, macOS, and Windows.
- Serves a browser-based control panel with config-backed username/password login.
- Refreshes, classifies, probes, and connects VPNGate nodes.
- Runs an in-process OpenVPN protocol engine and userspace TCP/IP stack; no OpenVPN installation, TUN driver, or administrator privilege is required.
- Runs VMESS and SOCKS in-process using Go libraries; no xray executable is required.
- Exposes protected VMESS, Quantumult X, and Clash subscription endpoints on a subscription-only listener.
- Ships as one Go binary with embedded HTML assets and protocol engines.
- Generates service templates for systemd, launchd, and Windows `sc.exe`.

## Quick Start

中文用户请直接阅读：[中文快速上手](docs/QUICK_START_ZH.md)。

```bash
go run ./cmd/rim config init --output config.json
go run ./cmd/rim serve --config config.json
```

Open:

```text
http://127.0.0.1:8899
```

`config init` prints a randomly generated Web password once. The username and password are stored in `config.json` and can be changed there.

On startup, the default gateway config attempts to:

1. Start the VMESS inbound.
2. Refresh and probe VPNGate nodes.
3. Connect OpenVPN to a usable VPNGate exit.

Gateway mode expects both `subscription.enabled` and `proxy_core.enabled` to stay on. That is what turns the machine into a shareable VMESS gateway instead of a private-only exit.

Local applications use `proxy_core.local_socks_listen` (default `127.0.0.1:1080`). Remote clients use the generated VMESS subscription. Both paths leave through the selected VPNGate node.

No external VPN/proxy software is required. A reachable VPNGate server and Internet access are required. Public VMESS deployments also need the configured TCP port opened by the host firewall/NAT and should place the Web console behind TLS when exposed remotely.

The program is an application-level proxy gateway. It does not replace the operating-system default route: local browsers and apps must explicitly use the local SOCKS listener, while remote clients use a generated VMESS subscription.

## Security

- Keep the Web console on `127.0.0.1` unless it is protected by TLS and access controls.
- Treat `config.json`, subscription URLs, VMESS UUIDs, and runtime data as secrets.
- Subscription tokens are carried in URLs; use HTTPS for any non-local deployment.
- VPNGate nodes are community-operated and untrusted. Do not use them for sensitive traffic without understanding that trust model.
- Review [SECURITY.md](SECURITY.md) before publishing a server to the Internet.

## Subscriptions

```text
GET http://<host>:8898/sub/vmess?token=<config.subscription.token>
GET http://<host>:8898/sub/quantumult-x?token=<config.subscription.token>
GET http://<host>:8898/sub/clash?token=<config.subscription.token>
```

The subscription-only listener is configured with `subscription.listen` and does not expose the Web console or management API. Every subscription format is protected by the configured token and points clients at the embedded VMESS inbound.

Set `subscription.host` to `auto` for local and LAN use. In `auto` mode the app prefers a LAN/private address first, then falls back to a public address when that is the only usable option. Set it to a public IP/domain when publishing the gateway outside your network.

## Build

```bash
sh scripts/build_go.sh
```

Windows:

```powershell
.\scripts\build_go.ps1
```

Artifacts are written to `dist/go/` and include self-contained binaries, per-platform packages, service templates, and `SHA256SUMS.txt`.

## Service Templates

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

## Delivery Notes

See [docs/GO_DELIVERY.md](docs/GO_DELIVERY.md) for the full release checklist, runtime topology, and smoke tests.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues must be reported privately according to [SECURITY.md](SECURITY.md).

## License

Residential IP Manager is distributed under GNU GPL version 3. Third-party components retain their respective licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
