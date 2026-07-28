# Residential IP Manager 中文快速上手

Residential IP Manager 是一个跨平台、单二进制的 VPNGate 网关。OpenVPN 协议、VMESS、SOCKS 和 Web 控制台都已经编译进程序，不需要另外安装 OpenVPN、xray、Python 或 TUN 驱动。

## 1. 选择启动方式

普通用户建议下载与系统匹配的 Release 压缩包：

- macOS Apple 芯片：`darwin-arm64`
- macOS Intel：`darwin-amd64`
- Windows 64 位：`windows-amd64`
- Linux x86_64：`linux-amd64`
- Linux ARM64：`linux-arm64`

开发者从源码运行需要 Go 1.26 或更高版本。

## 2. 第一次生成配置

### macOS / Linux

先给二进制执行权限，文件名按实际下载版本替换：

```bash
chmod +x rim-darwin-arm64
./rim-darwin-arm64 config init --output config.json
```

Linux 示例：

```bash
chmod +x rim-linux-amd64
./rim-linux-amd64 config init --output config.json
```

### Windows

在 PowerShell 中执行：

```powershell
.\rim-windows-amd64.exe config init --output config.json
```

初始化命令会随机生成：

- Web 控制台密码
- Session Secret
- 订阅 Token
- VMESS UUID

终端只会显示一次随机 Web 密码。账号默认为 `admin`，账号和密码也保存在 `config.json` 的 `auth` 部分。不要把这个配置文件提交到 Git 仓库或发送给其他人。

如果 `config.json` 已存在，程序会拒绝覆盖。继续使用原配置即可；确实需要重建时，应先自行备份并移走旧文件。

## 3. 启动程序

macOS / Linux：

```bash
./rim-darwin-arm64 serve --config config.json
```

Windows：

```powershell
.\rim-windows-amd64.exe serve --config config.json
```

源码启动：

```bash
go run ./cmd/rim config init --output config.json
go run ./cmd/rim serve --config config.json
```

看到下面两类日志表示控制面已经启动：

```text
listening addr=127.0.0.1:8899
listening addr=0.0.0.0:8898
```

然后打开：

```text
http://127.0.0.1:8899
```

使用 `config.json` 中的账号密码登录。

## 4. 判断是否真正连接成功

“程序已启动”和“VPNGate 出口已连接”是两件事。只有控制台同时显示以下状态，才表示代理链路真正可用：

- 状态为“出口已连接”
- 提示“OpenVPN 已连接，出口复核通过”
- 页面显示了公网出口 IP
- VMESS 服务显示运行中

还可以通过 SOCKS 代理验证出口：

```bash
curl --socks5-hostname 127.0.0.1:1080 https://api.ipify.org
```

命令返回的 IP 应与控制台显示的出口 IP 相同。

如果只是打开程序但没有连接成功，本机网站流量不会自动改变。程序提供的是应用级代理，不会修改整个操作系统的默认路由。

## 5. 本机应用如何使用

不使用 Quantumult X 也可以。把需要代理的浏览器或应用设置为：

```text
类型：SOCKS5
服务器：127.0.0.1
端口：1080
```

只有明确使用该 SOCKS5 代理的应用才会经过 VPNGate 出口。未配置代理的应用仍使用原网络。

## 6. Quantumult X 如何使用

1. 登录 Web 控制台。
2. 在“VMESS 订阅”区域选择“Quantumult X”。
3. 点击“复制订阅地址”。
4. 在 Quantumult X 中添加远程节点/资源，并粘贴订阅地址。
5. 更新资源后选择生成的 VMESS 节点。
6. 开启 Quantumult X，并访问公网 IP 检测网站验证出口。

订阅 URL 包含访问 Token，应当按密码处理，不要公开到 Issue、聊天截图或日志中。

## 7. 部署到服务器供别人使用

编辑 `config.json`：

- 把 `subscription.host` 设置为服务器公网 IP 或域名，不要使用 `auto`。
- 保持 VMESS 端口与 `subscription.port` 一致，默认是 `10086`。
- 默认订阅端口是 `8898`。
- Web 控制台 `server.listen` 建议继续使用 `127.0.0.1:8899`。

服务器防火墙和云安全组至少需要放行：

```text
TCP 10086  VMESS 客户端连接
TCP 8898   订阅地址
```

如果服务器在路由器或 NAT 后面，还需要把这两个端口映射到部署机器。公网订阅应配置 HTTPS 反向代理，因为 Token 位于 URL 中。不要直接把管理端口 `8899` 暴露到公网。

## 8. 如何关闭

前台运行时，在启动程序的终端按：

```text
Ctrl+C
```

程序会正常停止 Web、订阅、VMESS、SOCKS 和 VPNGate 隧道。默认端口全部释放后才算关闭完成。

检查端口的方法：

macOS / Linux：

```bash
lsof -nP -iTCP -sTCP:LISTEN | grep -E ':(8898|8899|10086|1080)'
```

Windows PowerShell：

```powershell
Get-NetTCPConnection -State Listen | Where-Object LocalPort -in 8898,8899,10086,1080
```

没有输出表示这些端口已经释放。

## 9. 常见问题

### 控制台打不开

确认终端中存在 `listening addr=127.0.0.1:8899`，并检查 `8899` 是否被其他程序占用。

### 节点很多但一直连接失败

VPNGate 是社区免费节点，节点可能随时离线、限速或拒绝连接。刷新节点并等待程序尝试其他候选。只有“出口复核通过”才能算成功。

### 关闭自己的代理后无法访问国外网站

这是预期行为。Residential IP Manager 不修改系统默认路由。本机浏览器必须使用 `127.0.0.1:1080` SOCKS5，或者通过 Quantumult X 使用生成的 VMESS 节点。

### Quantumult X 订阅可以更新但节点不能连接

检查控制台是否已经连接 VPNGate 出口，并确认 Quantumult X 能访问部署机器的 VMESS 端口。公网部署还需要检查服务器防火墙、云安全组、NAT 映射和 `subscription.host`。

### macOS 提示无法验证开发者

未签名的本地构建可能触发 Gatekeeper。优先使用项目正式签名的 Release；开发构建可在系统“隐私与安全性”中确认来源后手动允许。

## 10. 安全提醒

- VPNGate 节点由第三方运营，不应视为可信网络。
- 不要公开 `config.json`、订阅 URL、Token、VMESS UUID 或运行日志中的个人网络信息。
- 公网部署应使用主机防火墙和 HTTPS。
- 下载 Release 后应使用随包提供的 `SHA256SUMS.txt` 校验文件。
- 安全问题请按照项目根目录的 `SECURITY.md` 私下报告。
