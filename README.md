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

安装时会交互设置面板端口、多级路径、管理员用户名和密码，并选择使用 IP 或域名访问。域名模式会自动申请 Let's Encrypt 证书并开启续签。安装完成后使用 `slinx` 命令管理面板。

## 简介

SLINX node 是一个基于 sing-box 核心的轻量节点管理面板，专注入站、用户、端口转发、证书和网络端点管理。

## 功能

- 🚀 基于 sing-box 核心，支持 VLESS、VMess、Hysteria2、Trojan、TUIC、AnyTLS 和 Tunnel 转发
- 🎯 Reality SNI 目标扫描与可用性检查
- 📜 证书管理，支持 Let's Encrypt、ZeroSSL，DNS/HTTP 验证和自动续签
- 👥 多用户管理，单节点链接可复制或使用二维码分享
- 🌐 端点管理（WireGuard / Cloudflare WARP），支持一键注册换IP与路由规则绑定
- 🔍 IP 检测、解锁检测、回程检测
- 🔄 一键更新，支持核心独立更新

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

当前支持：VLESS、VMess、Hysteria2、Trojan、TUIC、AnyTLS、Tunnel（TCP/UDP 转发）、WireGuard（出站）

计划支持：Shadowsocks、ShadowTLS

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
