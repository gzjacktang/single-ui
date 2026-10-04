<div align="center">
  <img src="web/src/asset/image/logo.webp" width="50" />
  <h1>SLINX node</h1>
  <p>基于 sing-box 核心的节点管理面板</p>

	[![Release](https://img.shields.io/github/v/release/gzjacktang/single-ui)](https://github.com/gzjacktang/single-ui/releases)
  [![License](https://img.shields.io/badge/license-MIT-blue.svg?longCache=true)](LICENSE)
</div>

> [!IMPORTANT]
> 本项目仅供个人使用。请勿将其用于非法目的，也请勿在生产环境中使用。

## 安装

```bash
bash <(curl -sL https://raw.githubusercontent.com/gzjacktang/single-ui/main/install.sh)
```

安装脚本会交互设置：

- 面板监听端口（系统保留端口会自动拒绝）
- 面板路径（支持 `/private/admin/panel` 这样的多级路径）
- 管理员用户名和密码
- 使用服务器 IP 或域名访问

选择域名访问时，请先将域名解析到本机，并确保 TCP 80 端口可被公网访问。安装程序会自动注册 Let's Encrypt、申请面板证书、启用 HTTPS，并由后台任务自动续签。

安装完成后，使用 `slinx` 进入终端管理菜单。面板端口只在安装阶段设置，面板设置页面不会提供端口修改项。

### 旧版终端更新失败时

v0.0.7 的终端“更新”会尝试直接写入正在运行的程序文件。若显示“下载失败”，请在 VPS 的 root 终端执行一次手动更新；已有面板数据库和证书不会被重置：

```bash
arch=$(uname -m)
case "$arch" in
  x86_64) arch=amd64 ;;
  aarch64) arch=arm64 ;;
  *) echo "不支持的架构: $arch"; exit 1 ;;
esac
tmp=$(mktemp /etc/slinx/.slinx-update.XXXXXX) || exit 1
if ! curl -fL --retry 3 -o "$tmp" "https://github.com/gzjacktang/single-ui/releases/latest/download/slinx_linux_${arch}"; then
  rm -f "$tmp"
  exit 1
fi
test -s "$tmp" && chmod 755 "$tmp" && mv "$tmp" /etc/slinx/slinx && systemctl restart slinx
```

如果 `curl` 也失败，请先检查 VPS 到 GitHub Release 下载地址的连通性，并保留 `curl` 输出以便定位；下载失败不会替换现有程序。

### 手动初始化

安装脚本会调用内置的 `setup` 命令。需要手动初始化时，可以使用：

```bash
cd /etc/slinx
SLINX_SETUP_PASSWORD='your-password' ./slinx setup \
  --port 2053 \
  --path /private/admin/panel \
  --username admin_user \
  --access ip
```

域名模式还需要增加 `--domain` 和 `--email` 参数。用户名和密码至少 6 位；密码通过环境变量传入，不会出现在命令参数列表中。

## 简介

SLINX node 是一个基于 sing-box 核心的轻量节点管理面板，专注入站、用户、端口转发、证书和网络端点管理。

## 功能

- 🚀 基于 sing-box 核心，支持 VLESS、VMess、Hysteria2、Trojan、TUIC、AnyTLS、Shadowsocks 2022 和 Tunnel 转发
- 🎯 Reality SNI 目标扫描与可用性检查，支持域名、IP 和 CIDR 网段，自动过滤内网地址并从证书发现可用 SNI
- 📜 证书管理，支持 Let's Encrypt、ZeroSSL，DNS/HTTP 验证和自动续签
- 👥 多用户管理，单节点链接可复制或使用二维码分享
- 🌐 端点管理（WireGuard / Cloudflare WARP），支持一键注册换IP与路由规则绑定
- 🔍 IP 检测、解锁检测、回程检测
- 🔄 一键更新，支持核心独立更新

## 功能边界

- 不提供机器资源监控仪表盘，也不会启动 CPU、内存、流量等监控任务。
- 不提供订阅服务、订阅端口或订阅页面；用户分享使用单节点链接、复制和二维码。
- 不提供面板对接和同步功能。
- 管理员可以在“面板设置”中修改用户名和密码；修改路径后需要按页面提示重启面板。

## 关于本项目

SLINX node 专为普通自建节点用户设计，化繁为简，让搭建节点不再是门槛。

整个使用流程极其简单：登录终端 → 运行安装脚本完成初始设置 → 打开面板 → 添加入站和用户 → 复制节点链接。

本项目不追求功能堆砌，只做普通用户真正需要的功能。

## 截图

![Detection](doc/detect.png)
![Inbound](doc/inbound.png)

## 支持的平台

**操作系统：** Ubuntu、Debian、Oracle Linux

**架构：** `amd64` · `arm64`

## 协议

当前支持：VLESS、VMess、Hysteria2、Trojan、TUIC、AnyTLS、Shadowsocks、Tunnel（TCP/UDP 转发）、WireGuard（出站）。Shadowsocks 可选 `2022-blake3-aes-256-gcm`、`aes-256-gcm`、`chacha20-ietf-poly1305`、`xchacha20-ietf-poly1305`；不支持 `chacha20-poly1305` 这个非 IETF 名称。入站需关联至少一位启用用户；每位用户使用独立的 32 字节随机密钥，可复制 `ss://` 链接或二维码分享。2022 模式另外需要服务端身份密钥，并要求客户端支持 Shadowsocks 2022。

计划支持：ShadowTLS

## 多语言支持

当前支持：简体中文

v3 版本计划支持：繁体中文、English、Русский、فارسی

## 社区

[![Telegram](https://img.shields.io/badge/Telegram-@SLINXlink-26A5E4?logo=telegram&logoColor=white)](https://t.me/slinxlink)

## 鸣谢

- [sing-box](https://github.com/SagerNet/sing-box) — 核心代理引擎
- [3x-ui](https://github.com/MHSanaei/3x-ui) — UI 界面设计借鉴
- [lmc999](https://github.com/lmc999/RegionRestrictionCheck) — 解锁检测脚本，已进行 Go 语言化改造
- [Loyalsoldier](https://github.com/Loyalsoldier/clash-rules) — Clash 规则集
- [ACL4SSR](https://github.com/ACL4SSR/ACL4SSR) — Clash 规则集
- [Claude](https://claude.ai) — AI 编程助手，本项目大量代码由 Claude 协助完成

## 友情链接

- [LINUX DO](https://linux.do)

## License

[MIT](LICENSE)
