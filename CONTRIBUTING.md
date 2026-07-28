# Contributing

## Development Setup

Install Go 1.26 or newer. No Python, OpenVPN, xray executable, TUN driver, or JavaScript toolchain is required.

```bash
go test ./...
(cd third_party/minivpn && go test ./...)
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
(cd third_party/minivpn && go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...)
sh scripts/build_go.sh
```

Run `gofmt` on changed Go files before opening a pull request. Keep changes scoped, add tests for behavior changes, and test platform-specific behavior on the affected operating system.

## Pull Requests

- Explain the problem, behavior change, and verification performed.
- Do not commit generated configs, runtime state, subscriptions, credentials, logs, databases, OpenVPN profiles, or build artifacts.
- Preserve the in-process, single-binary runtime model unless a design change is discussed first.
- Changes to `third_party/minivpn` must update `third_party/minivpn/LOCAL_CHANGES.md` and remain GPL-3.0 compatible.
- UI changes should include desktop and mobile screenshots.

## Security

Do not report vulnerabilities in public issues. Follow `SECURITY.md` and use GitHub Security Advisories.
