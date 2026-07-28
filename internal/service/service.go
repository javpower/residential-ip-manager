package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Options struct {
	BinaryPath string
	ConfigPath string
	DataDir    string
	Listen     string
	Version    string
}

func Generate(platform string, opts Options, outputDir string) error {
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	switch strings.ToLower(platform) {
	case "linux":
		return os.WriteFile(
			filepath.Join(outputDir, "residential-ip-manager.service"),
			[]byte(systemdUnit(opts)),
			0o644,
		)
	case "darwin", "macos":
		return os.WriteFile(
			filepath.Join(outputDir, "com.guli-joy.residential-ip-manager.plist"),
			[]byte(launchdPlist(opts)),
			0o644,
		)
	case "windows":
		if err := os.WriteFile(filepath.Join(outputDir, "install-service.ps1"), []byte(windowsInstallScript(opts)), 0o644); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(outputDir, "uninstall-service.ps1"), []byte(windowsUninstallScript()), 0o644)
	default:
		return fmt.Errorf("unsupported platform %q", platform)
	}
}

func systemdUnit(opts Options) string {
	binary := valueOr(opts.BinaryPath, "/usr/local/bin/rim")
	configPath := valueOr(opts.ConfigPath, "/etc/residential-ip-manager/config.json")
	dataDir := valueOr(opts.DataDir, "/var/lib/residential-ip-manager")
	listen := valueOr(opts.Listen, "127.0.0.1:8899")
	return fmt.Sprintf(`[Unit]
Description=Residential IP Manager Go
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s serve --config %s --listen %s
WorkingDirectory=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, binary, configPath, listen, dataDir)
}

func launchdPlist(opts Options) string {
	binary := valueOr(opts.BinaryPath, "/usr/local/bin/rim")
	configPath := valueOr(opts.ConfigPath, "/usr/local/etc/residential-ip-manager/config.json")
	dataDir := valueOr(opts.DataDir, "/usr/local/var/residential-ip-manager")
	listen := valueOr(opts.Listen, "127.0.0.1:8899")
	label := "com.guli-joy.residential-ip-manager"
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>serve</string>
    <string>--config</string>
    <string>%s</string>
    <string>--listen</string>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>WorkingDirectory</key>
  <string>%s</string>
  <key>EnvironmentVariables</key>
  <dict>
  </dict>
</dict>
</plist>
`, label, binary, configPath, listen, dataDir)
}

func windowsInstallScript(opts Options) string {
	binary := valueOr(opts.BinaryPath, "rim.exe")
	configPath := valueOr(opts.ConfigPath, "$env:ProgramData\\ResidentialIPManager\\config.json")
	serviceName := "ResidentialIPManagerGo"
	return fmt.Sprintf(
		"$ErrorActionPreference = \"Stop\"\n"+
			"$binary = \"%s\"\n"+
			"$config = \"%s\"\n"+
			"$listen = \"%s\"\n"+
			"$name = \"%s\"\n\n"+
			"if (-not (Get-Command sc.exe -ErrorAction SilentlyContinue)) {\n"+
			"    throw \"sc.exe is required\"\n"+
			"}\n\n"+
			"$binPath = \"`\"$binary`\" serve --config `\"$config`\" --listen `\"$listen`\"\"\n"+
			"sc.exe create $name binPath= $binPath start= auto | Out-Null\n"+
			"sc.exe description $name \"Residential IP Manager Go\" | Out-Null\n"+
			"Write-Host \"Installed $name\"\n",
		binary,
		configPath,
		valueOr(opts.Listen, "127.0.0.1:8899"),
		serviceName,
	)
}

func windowsUninstallScript() string {
	return "$ErrorActionPreference = \"Stop\"\n" +
		"$name = \"ResidentialIPManagerGo\"\n" +
		"sc.exe stop $name | Out-Null\n" +
		"sc.exe delete $name | Out-Null\n" +
		"Write-Host \"Removed ResidentialIPManagerGo\"\n"
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func DefaultBinaryPath() string {
	if runtime.GOOS == "windows" {
		return "rim.exe"
	}
	return "rim"
}
