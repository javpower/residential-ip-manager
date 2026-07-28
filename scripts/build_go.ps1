param(
    [string]$OutDir = "$PSScriptRoot\..\dist\go",
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"
$RootDir = Resolve-Path "$PSScriptRoot\.."
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $OutDir "packages") | Out-Null
if ($env:GO_BUILD_CACHE) {
    New-Item -ItemType Directory -Force -Path $env:GO_BUILD_CACHE | Out-Null
    $env:GOCACHE = $env:GO_BUILD_CACHE
}

function Build-One {
    param(
        [string]$OS,
        [string]$Arch,
        [string]$Ext
    )
    $name = "rim-$OS-$Arch$Ext"
    Write-Host "building $name"
    Push-Location $RootDir
    try {
        $env:GOOS = $OS
        $env:GOARCH = $Arch
        $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $OutDir $name) ./cmd/rim
    }
    finally {
        Pop-Location
        Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
    }
}

Build-One linux amd64 ""
Build-One linux arm64 ""
Build-One windows amd64 ".exe"
Build-One darwin amd64 ""
Build-One darwin arm64 ""

Push-Location $RootDir
try {
    $hostOS = (& go env GOHOSTOS).Trim()
    $hostArch = (& go env GOHOSTARCH).Trim()
    $env:GOOS = $hostOS
    $env:GOARCH = $hostArch
    go run ./cmd/rim service generate --platform linux --output (Join-Path $OutDir "service/linux") --binary "/usr/local/bin/rim" --config "/etc/residential-ip-manager/config.json" --data-dir "/var/lib/residential-ip-manager"
    go run ./cmd/rim service generate --platform darwin --output (Join-Path $OutDir "service/darwin") --binary "/usr/local/bin/rim" --config "/usr/local/etc/residential-ip-manager/config.json" --data-dir "/usr/local/var/residential-ip-manager"
    go run ./cmd/rim service generate --platform windows --output (Join-Path $OutDir "service/windows") --binary "C:\Program Files\ResidentialIPManager\rim.exe" --config "C:\ProgramData\ResidentialIPManager\config.json"
}
finally {
    Pop-Location
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
}

function Package-One {
    param(
        [string]$OS,
        [string]$Arch,
        [string]$Ext,
        [string]$ServicePlatform
    )
    $binary = "rim-$OS-$Arch$Ext"
    $packageName = "residential-ip-manager-$Version-$OS-$Arch"
    $packagesDir = Join-Path $OutDir "packages"
    $packageDir = Join-Path $packagesDir $packageName
    Remove-Item -Recurse -Force $packageDir -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $packageDir | Out-Null
    Copy-Item (Join-Path $OutDir $binary) $packageDir
    Copy-Item (Join-Path $RootDir "README.md") $packageDir
    Copy-Item (Join-Path $RootDir "SECURITY.md") $packageDir
    Copy-Item (Join-Path $RootDir "LICENSE") $packageDir
    Copy-Item (Join-Path $RootDir "THIRD_PARTY_NOTICES.md") $packageDir
    New-Item -ItemType Directory -Force -Path (Join-Path $packageDir "configs") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $packageDir "docs") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $packageDir "licenses") | Out-Null
    Copy-Item (Join-Path $RootDir "configs/go.example.json") (Join-Path $packageDir "configs")
    Copy-Item (Join-Path $RootDir "docs/GO_DELIVERY.md") (Join-Path $packageDir "docs")
    Copy-Item (Join-Path $RootDir "licenses/XRAY-MPL-2.0.txt") (Join-Path $packageDir "licenses")
    Copy-Item (Join-Path $RootDir "third_party/minivpn/LOCAL_CHANGES.md") (Join-Path $packageDir "licenses/MINIVPN-LOCAL-CHANGES.md")
    $serviceDir = Join-Path $OutDir "service/$ServicePlatform"
    if (Test-Path $serviceDir) {
        New-Item -ItemType Directory -Force -Path (Join-Path $packageDir "service") | Out-Null
        Copy-Item "$serviceDir/*" (Join-Path $packageDir "service") -Recurse
    }
    $archive = Join-Path $packagesDir "$packageName.zip"
    Remove-Item -Force $archive -ErrorAction SilentlyContinue
    Compress-Archive -Path $packageDir -DestinationPath $archive
}

Package-One linux amd64 "" linux
Package-One linux arm64 "" linux
Package-One windows amd64 ".exe" windows
Package-One darwin amd64 "" darwin
Package-One darwin arm64 "" darwin

$packageFiles = Get-ChildItem (Join-Path $OutDir "packages") -File |
    Where-Object { $_.Name.EndsWith(".zip") -or $_.Name.EndsWith(".tar.gz") }
@(Get-ChildItem $OutDir -Filter "rim-*") + @($packageFiles) |
    Get-FileHash -Algorithm SHA256 |
    ForEach-Object {
        $relative = Resolve-Path -Relative $_.Path
        "$($_.Hash.ToLower())  $relative"
    } |
    Set-Content -Encoding ascii (Join-Path $OutDir "SHA256SUMS.txt")

Write-Host "artifacts written to $OutDir"
