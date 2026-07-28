#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT_DIR="${OUT_DIR:-$ROOT_DIR/dist/go}"
VERSION="${VERSION:-dev}"

mkdir -p "$OUT_DIR"
mkdir -p "$OUT_DIR/packages"
if [ -n "${GO_BUILD_CACHE:-}" ]; then
  mkdir -p "$GO_BUILD_CACHE"
  export GOCACHE="$GO_BUILD_CACHE"
fi

build_one() {
  os="$1"
  arch="$2"
  ext="$3"
  name="rim-${os}-${arch}${ext}"
  echo "building $name"
  (
    cd "$ROOT_DIR"
    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w -X main.version=$VERSION" \
      -o "$OUT_DIR/$name" ./cmd/rim
  )
}

build_one linux amd64 ""
build_one linux arm64 ""
build_one windows amd64 ".exe"
build_one darwin amd64 ""
build_one darwin arm64 ""

(
  cd "$ROOT_DIR"
  host_os=$(go env GOHOSTOS)
  host_arch=$(go env GOHOSTARCH)
  GOOS="$host_os" GOARCH="$host_arch" go run ./cmd/rim service generate --platform linux --output "$OUT_DIR/service/linux" \
    --binary /usr/local/bin/rim \
    --config /etc/residential-ip-manager/config.json \
    --data-dir /var/lib/residential-ip-manager
  GOOS="$host_os" GOARCH="$host_arch" go run ./cmd/rim service generate --platform darwin --output "$OUT_DIR/service/darwin" \
    --binary /usr/local/bin/rim \
    --config /usr/local/etc/residential-ip-manager/config.json \
    --data-dir /usr/local/var/residential-ip-manager
  GOOS="$host_os" GOARCH="$host_arch" go run ./cmd/rim service generate --platform windows --output "$OUT_DIR/service/windows" \
    --binary 'C:\Program Files\ResidentialIPManager\rim.exe' \
    --config 'C:\ProgramData\ResidentialIPManager\config.json'
)

package_one() {
  os="$1"
  arch="$2"
  ext="$3"
  service_platform="$4"
  binary="rim-${os}-${arch}${ext}"
  package_name="residential-ip-manager-${VERSION}-${os}-${arch}"
  package_dir="$OUT_DIR/packages/$package_name"
  rm -rf "$package_dir"
  mkdir -p "$package_dir"
  cp "$OUT_DIR/$binary" "$package_dir/"
  cp "$ROOT_DIR/README.md" "$package_dir/"
  cp "$ROOT_DIR/SECURITY.md" "$package_dir/"
  cp "$ROOT_DIR/LICENSE" "$package_dir/"
  cp "$ROOT_DIR/THIRD_PARTY_NOTICES.md" "$package_dir/"
  mkdir -p "$package_dir/configs" "$package_dir/docs" "$package_dir/licenses"
  cp "$ROOT_DIR/configs/go.example.json" "$package_dir/configs/"
  cp "$ROOT_DIR/docs/GO_DELIVERY.md" "$package_dir/docs/"
  cp "$ROOT_DIR/licenses/XRAY-MPL-2.0.txt" "$package_dir/licenses/"
  cp "$ROOT_DIR/third_party/minivpn/LOCAL_CHANGES.md" "$package_dir/licenses/MINIVPN-LOCAL-CHANGES.md"
  if [ -d "$OUT_DIR/service/$service_platform" ]; then
    mkdir -p "$package_dir/service"
    cp -R "$OUT_DIR/service/$service_platform/." "$package_dir/service/"
  fi
  (
    cd "$OUT_DIR/packages"
    if [ "$os" = "windows" ] && command -v zip >/dev/null 2>&1; then
      zip -qr "$package_name.zip" "$package_name"
    else
      tar -czf "$package_name.tar.gz" "$package_name"
    fi
  )
}

package_one linux amd64 "" linux
package_one linux arm64 "" linux
package_one windows amd64 ".exe" windows
package_one darwin amd64 "" darwin
package_one darwin arm64 "" darwin

(
  cd "$OUT_DIR"
  if command -v sha256sum >/dev/null 2>&1; then
    find . -maxdepth 2 -type f \( -name 'rim-*' -o -name '*.tar.gz' -o -name '*.zip' \) -print | sort | xargs sha256sum > SHA256SUMS.txt
  elif command -v shasum >/dev/null 2>&1; then
    find . -maxdepth 2 -type f \( -name 'rim-*' -o -name '*.tar.gz' -o -name '*.zip' \) -print | sort | xargs shasum -a 256 > SHA256SUMS.txt
  else
    echo "warning: no SHA256 tool found" >&2
  fi
)

echo "artifacts written to $OUT_DIR"
