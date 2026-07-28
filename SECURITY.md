# Security Policy

## 支持范围

安全修复优先应用于最新 Release 和 `main` 分支。本项目会处理代理流量、OpenVPN 配置和
本机代理流量，因此请只从本仓库 Releases 下载构建产物，并核对随 Release 提供的
`SHA256SUMS.txt`。

## 报告安全问题

请通过 GitHub Security Advisory 的 **Report a vulnerability** 私密报告安全问题，不要在
公开 Issue 中提交订阅地址、VMESS UUID、Web 凭据、完整日志或可识别个人网络的信息。

报告建议包含：

- 受影响版本与操作系统/平台版本；
- 可复现的最小步骤；
- 预期行为与实际行为；
- 已脱敏的日志或配置片段；
- 对路由、DNS、凭据或进程所有权的实际影响。

## 信任边界

- VPNGate 节点和远程 OpenVPN 配置均按不可信输入处理；
- 程序不会保证公共 VPN 节点的运营者、内容或长期可用性；
- IP 类型判断依赖第三方情报，只能降低误判，不能提供永久的住宅属性保证；
- Go 版的 `auth.password`、`server.session_secret`、`subscription.token`、VMESS UUID、
  状态文件和 OpenVPN 临时配置都应视为敏感信息；
- 如果把 Web 控制台或 VMESS 端口监听到非 localhost 地址，请先配置主机防火墙，并使用
  可信的反向代理或等效访问控制。
- 订阅令牌位于 URL 查询参数中。公网部署必须使用 HTTPS，并避免把完整订阅 URL 写入代理、
  分析平台或公开日志。
