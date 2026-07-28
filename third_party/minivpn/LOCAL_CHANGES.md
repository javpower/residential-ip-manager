# Local Changes to minivpn

Base: `github.com/ooni/minivpn` v0.0.7.

Residential IP Manager carries a modified source copy under `third_party/minivpn` so the complete corresponding source is available with every project release.

## 2026-07-28

- Added legacy `P_DATA_V1` packet parsing, serialization, and data-channel selection when a peer ID is not negotiated.
- Added VPNGate/SoftEther-compatible control-message and early-data handling.
- Ignored SoftEther keepalive payloads that are not IP packets.
- Exposed negotiated tunnel information needed by the in-process userspace network stack.
- Added focused tests for the packet formats and compatibility behavior above.
- Removed the unused upstream Docker integration-test entrypoint; product and protocol unit tests remain in place, while obsolete Moby/runc test-only dependencies are no longer part of this source module.
- Upgraded uTLS to the Xray-compatible 2026 revision and adapted certificate/tests for Go 1.26 security requirements.

Modified implementation areas are under `internal/datachannel`, `internal/model`, `internal/packetmuxer`, `internal/session`, `internal/tlssession`, and `internal/tun`.

All local modifications remain licensed under GNU GPL version 3. See `LICENSE` in this directory.
