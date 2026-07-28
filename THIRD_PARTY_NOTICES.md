# Third-Party Notices

Residential IP Manager statically links third-party Go modules listed in `go.mod` and `go.sum`.

The in-process OpenVPN implementation is derived from `github.com/ooni/minivpn` v0.0.7 and is included under `third_party/minivpn`. minivpn is licensed under GNU GPL version 3. Local modifications add VPNGate/SoftEther `P_DATA_V1` interoperability, early-data handling, and userspace-netstack integration support. See `third_party/minivpn/LOCAL_CHANGES.md`.

The embedded VMESS implementation is provided by `github.com/xtls/xray-core` v1.260327.0 under the Mozilla Public License 2.0. The license text is included at `licenses/XRAY-MPL-2.0.txt`.

Additional dependency license texts and copyright notices remain in their respective upstream modules. Release source is the authoritative corresponding source for the linked binary.
