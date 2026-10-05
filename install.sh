#!/bin/bash

set -e

CUR_DIR=$(pwd)

RED='\033[0;31m'
PINK='\033[38;2;240;128;128m'
PLAIN='\033[0m'

step() {
    echo ""
    echo -e "${PINK}>>> $1${PLAIN}"
    echo -e "${PINK}————————————————————————————————————————${PLAIN}"
}

# 检查 root
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}请以 root 用户运行此脚本${PLAIN}"
    exit 1
fi

FRESH_INSTALL=false
if [[ ! -f /etc/slinx/data/slinx.db ]]; then
    FRESH_INSTALL=true
    step "设置面板"

    while true; do
        read -rp "面板端口 [2053]: " PANEL_PORT
        PANEL_PORT=${PANEL_PORT:-2053}
        if [[ $PANEL_PORT =~ ^[0-9]+$ ]] && (( PANEL_PORT >= 1 && PANEL_PORT <= 65535 )); then
            case "$PANEL_PORT" in
                21|22|23|25|53|80|110|143|465|587|993|995|2048|3306|3389|5432|6379|9090)
                    echo -e "${RED}该端口为系统保留端口，请选择其他端口${PLAIN}"
                    ;;
                *) break ;;
            esac
            continue
        fi
        echo -e "${RED}端口必须在 1-65535 之间${PLAIN}"
    done

    DEFAULT_PATH="/panel/$(od -An -N4 -tx1 /dev/urandom | tr -d ' \n')"
    while true; do
        read -rp "面板路径 [$DEFAULT_PATH]: " PANEL_PATH
        PANEL_PATH=${PANEL_PATH:-$DEFAULT_PATH}
        if [[ $PANEL_PATH =~ ^/([A-Za-z0-9._~-]+/)*[A-Za-z0-9._~-]+$ ]]; then
            break
        fi
        echo -e "${RED}路径必须以 / 开头，可使用多级路径${PLAIN}"
    done

    while true; do
        read -rp "管理员用户名: " PANEL_USERNAME
        if [[ $PANEL_USERNAME =~ ^[A-Za-z0-9_]{6,}$ ]]; then
            break
        fi
        echo -e "${RED}用户名至少 6 位，只能包含字母、数字和下划线${PLAIN}"
    done

    while true; do
        read -rsp "管理员密码: " PANEL_PASSWORD
        echo ""
        read -rsp "再次输入密码: " PANEL_PASSWORD_CONFIRM
        echo ""
        if [[ ${#PANEL_PASSWORD} -ge 6 && "$PANEL_PASSWORD" == "$PANEL_PASSWORD_CONFIRM" ]]; then
            break
        fi
        echo -e "${RED}密码至少 6 位，两次输入必须一致${PLAIN}"
    done

    while true; do
        read -rp "访问方式 [1=IP, 2=域名]: " ACCESS_CHOICE
        case "$ACCESS_CHOICE" in
            1)
                ACCESS_MODE="ip"
                break
                ;;
            2)
                ACCESS_MODE="domain"
                read -rp "面板域名（需已解析到本机）: " PANEL_DOMAIN
                read -rp "Let's Encrypt 邮箱: " ACME_EMAIL
                break
                ;;
            *) echo -e "${RED}请输入 1 或 2${PLAIN}" ;;
        esac
    done
fi

# 检测包管理器
if command -v apt &>/dev/null; then
    PKG="apt"
elif command -v dnf &>/dev/null; then
    PKG="dnf"
elif command -v yum &>/dev/null; then
    PKG="yum"
else
    echo -e "${RED}不支持的包管理器${PLAIN}"
    exit 1
fi

step "更新系统"
$PKG update -y

step "安装依赖"
$PKG install -y curl chrony gzip sqlite3

step "设置系统时间同步"
timedatectl set-timezone Asia/Shanghai
systemctl enable chrony
systemctl restart chrony

step "检测 CPU 架构"
ARCH=$(uname -m)
if [[ $ARCH == "x86_64" ]]; then
    SLINX_ARCH="amd64"
elif [[ $ARCH == "aarch64" ]]; then
    SLINX_ARCH="arm64"
else
    echo -e "${RED}不支持的 CPU 架构: $ARCH${PLAIN}"
    exit 1
fi

step "检测最新版本"
RELEASE=$(curl -s "https://api.github.com/repos/gzjacktang/single-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$RELEASE" ]]; then
    echo -e "${RED}获取版本号失败${PLAIN}"
    exit 1
fi
echo -e "${PINK}最新版本: ${RELEASE}${PLAIN}"

step "检测最新 sing-box 核心版本"
CORE_VERSION=$(curl -s "https://api.github.com/repos/gzjacktang/single-ui/releases" | grep '"tag_name"' | grep 'sing-box-' | head -1 | sed 's/.*"sing-box-\(.*\)".*/\1/')
if [[ -z "$CORE_VERSION" ]]; then
    echo -e "${RED}获取 sing-box 版本号失败${PLAIN}"
    exit 1
fi
echo -e "${PINK}sing-box 版本: ${CORE_VERSION}${PLAIN}"

step "创建目录"
SLINX_DIR="/etc/slinx"
BIN_DIR="$SLINX_DIR/bin"
DATA_DIR="$SLINX_DIR/data"
CERT_DIR="$SLINX_DIR/cert"
mkdir -p $BIN_DIR $DATA_DIR $CERT_DIR

step "下载 SBOX 面板"
systemctl stop slinx.service 2>/dev/null || true
SLINX_URL="https://github.com/gzjacktang/single-ui/releases/download/${RELEASE}/slinx_linux_${SLINX_ARCH}"
curl -fLo $SLINX_DIR/slinx $SLINX_URL
chmod +x $SLINX_DIR/slinx

step "下载 sing-box"
SINGBOX_URL="https://github.com/gzjacktang/single-ui/releases/download/sing-box-${CORE_VERSION}/sing-box_linux_${SLINX_ARCH}.gz"
curl -fLo /tmp/sing-box.gz $SINGBOX_URL
gunzip /tmp/sing-box.gz
mv /tmp/sing-box $BIN_DIR/sing-box
chmod +x $BIN_DIR/sing-box

if [[ "$FRESH_INSTALL" == "true" ]]; then
    step "初始化面板"
    SETUP_ARGS=(setup --port "$PANEL_PORT" --path "$PANEL_PATH" --username "$PANEL_USERNAME" --access "$ACCESS_MODE")
    if [[ "$ACCESS_MODE" == "domain" ]]; then
        SETUP_ARGS+=(--domain "$PANEL_DOMAIN" --email "$ACME_EMAIL")
    fi
	if ! (cd "$SLINX_DIR" && SBOX_SETUP_PASSWORD="$PANEL_PASSWORD" ./slinx "${SETUP_ARGS[@]}"); then
		echo -e "${RED}面板初始化失败，已清理未完成的配置${PLAIN}"
		rm -f "$DATA_DIR/slinx.db" "$DATA_DIR/slinx.db-shm" "$DATA_DIR/slinx.db-wal"
		rm -rf "$CERT_DIR"
		mkdir -p "$CERT_DIR"
		exit 1
	fi
    unset PANEL_PASSWORD PANEL_PASSWORD_CONFIRM SBOX_SETUP_PASSWORD
fi

step "创建 systemd 服务"
cat <<EOF > /etc/systemd/system/slinx.service
[Unit]
Description=SBOX Service
After=network.target nss-lookup.target
Wants=network.target

[Service]
User=root
Group=root
Type=simple
LimitNOFILE=999999
WorkingDirectory=$SLINX_DIR
ExecStart=$SLINX_DIR/slinx
Restart=on-failure
RestartSec=5
StartLimitInterval=100s
StartLimitBurst=3

[Install]
WantedBy=multi-user.target
EOF

step "启动服务"
systemctl daemon-reload
systemctl enable slinx.service
systemctl start slinx.service

step "注册 SBOX 管理命令"
cat <<EOF > /usr/local/bin/sbox
#!/bin/bash
/etc/slinx/slinx cli
EOF
chmod +x /usr/local/bin/sbox

# 保留旧命令，避免已有自动化脚本在升级后失效。
cat <<EOF > /usr/local/bin/slinx
#!/bin/bash
/etc/slinx/slinx cli
EOF
chmod +x /usr/local/bin/slinx

echo ""
echo -e "${PINK}>>> 安装完成${PLAIN}"
echo -e "${PINK}————————————————————————————————————————${PLAIN}"
echo -e "${PINK}管理脚本命令: sbox${PLAIN}"
echo ""
sleep 2
sbox
