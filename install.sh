#!/bin/bash

set -e

REPO="hakimi-x/Orange-Service"
INSTALL_DIR="/opt/orange-service"
SERVICE_NAME="orange-service"
BINARY_NAME="orange-service"

echo "🚀 Orange Service Installer"

# 获取最新版本
LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "❌ Failed to get latest version"
    exit 1
fi
echo "📦 Latest version: $LATEST"

# 检查是否已安装
IS_UPDATE=false
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    IS_UPDATE=true
    echo "📋 Existing installation detected"
fi

# 如果是更新，先停止服务并删除旧二进制
if [ "$IS_UPDATE" = true ]; then
    if systemctl is-active --quiet $SERVICE_NAME 2>/dev/null; then
        echo "⏹️ Stopping service..."
        sudo systemctl stop $SERVICE_NAME
    fi
    echo "🗑️ Removing old binary..."
    sudo rm -f "$INSTALL_DIR/$BINARY_NAME"
fi

# 创建目录
sudo mkdir -p $INSTALL_DIR
cd $INSTALL_DIR

# 下载新版本
echo "⬇️ Downloading..."
sudo curl -L -o $BINARY_NAME "https://github.com/${REPO}/releases/download/${LATEST}/orange-service-linux-amd64"
sudo chmod +x $BINARY_NAME

# 创建配置文件（仅首次安装）
if [ ! -f config.yaml ]; then
    echo "📝 Creating config file..."
    sudo tee config.yaml > /dev/null << 'EOF'
server:
  port: 8001
  host: "127.0.0.1"
  base_url: "https://your-domain.com"

github:
  token: ""
  repo: "owner/repo"
  webhook_secret: ""

cache:
  dir: "github_cache"
EOF
fi

# 创建/更新 systemd 服务
echo "🔧 Configuring systemd service..."
sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null << EOF
[Unit]
Description=Orange Service
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
Environment=GOMEMLIMIT=64MiB
Environment=GOGC=50
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME

# 完成提示
if [ "$IS_UPDATE" = true ]; then
    echo "🔄 Starting service..."
    sudo systemctl start $SERVICE_NAME
    echo "✅ Update to $LATEST complete!"
else
    echo "✅ Installation complete!"
    echo ""
    echo "Next steps:"
    echo "  1. Edit config: sudo nano $INSTALL_DIR/config.yaml"
    echo "  2. Start service: sudo systemctl start $SERVICE_NAME"
fi

echo "  Check status: sudo systemctl status $SERVICE_NAME"
