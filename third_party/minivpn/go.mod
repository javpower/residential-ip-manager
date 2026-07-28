module github.com/ooni/minivpn

go 1.26

// pinning for backwards-incompatible change
// replace gitlab.com/yawning/obfs4.git v0.0.0-20220204003609-77af0cba934d => gitlab.com/yawning/obfs4.git v0.0.0-20210511220700-e330d1b7024b

require (
	git.torproject.org/pluggable-transports/goptlib.git v1.3.0
	github.com/Doridian/water v1.6.1
	github.com/apex/log v1.9.0
	github.com/google/go-cmp v0.5.9
	github.com/google/gopacket v1.1.19
	github.com/google/martian v2.1.0+incompatible
	github.com/google/uuid v1.6.0
	github.com/jackpal/gateway v1.0.11 // pinned to a previous version until we can use go1.21
	github.com/refraction-networking/utls v1.8.3-0.20260301010127-aa6edf4b11af
	gitlab.com/yawning/obfs4.git v0.0.0-20220904064028-336a71d6e4cf
	golang.org/x/net v0.38.0
	golang.org/x/sync v0.6.0
	golang.zx2c4.com/wireguard v0.0.0-20231211153847-12269c276173 // indirect
)

require golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8

require (
	filippo.io/edwards25519 v1.0.0-rc.1.0.20210721174708-390f27c3be20 // indirect
	github.com/Doridian/gopacket v1.2.1 // indirect
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	gitlab.com/yawning/edwards25519-extra.git v0.0.0-20211229043746-2f91fcc9fbdb // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
