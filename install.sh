#!/bin/bash

set -e

REPO="hakimi-x/Orange-Service"
INSTALL_DIR="/opt/orange-service"
SERVICE_NAME="orange-service"

echo "🚀 Installing Orange Service..."

# 获取最新版本
LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "❌ Failed to get latest version"
    exit 1
fi
echo "📦 Latest version: $LATEST"

# 创建目录
sudo mkdir -p $INSTALL_DIR
cd $INSTALL_DIR

# 下载
echo "⬇️ Downloading..."
sudo curl -L -o orange-service "https://github.com/${REPO}/releases/download/${LATEST}/orange-service-linux-amd64"
sudo chmod +x orange-service

# 创建配置文件
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
    echo "⚠️ Please edit $INSTALL_DIR/config.yaml"
fi

# 创建 systemd 服务
echo "🔧 Creating systemd service..."
sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null << EOF
[Unit]
Description=Orange Service
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/orange-service
Environment=GOMEMLIMIT=64MiB
Environment=GOGC=50
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME

# 如果服务已运行则重启
if systemctl is-active --quiet $SERVICE_NAME; then
    echo "🔄 Restarting service..."
    sudo systemctl restart $SERVICE_NAME
    echo "✅ Update complete!"
else
    echo "✅ Installation complete!"
    echo ""
    echo "Next steps:"
    echo "1. Edit config: sudo nano $INSTALL_DIR/config.yaml"
    echo "2. Start service: sudo systemctl start $SERVICE_NAME"
fi

echo "3. Check status: sudo systemctl status $SERVICE_NAME"
