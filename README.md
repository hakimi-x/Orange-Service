# Orange Service

GitHub Release 更新检查服务，为 Orange 客户端提供版本检查和下载服务。

## 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/hakimi-x/Orange-Service/main/install.sh | bash
```

## 功能

- 检查客户端更新
- 缓存 GitHub Release 资源
- Webhook 回调自动刷新版本

## 手动安装

1. 下载可执行文件
2. 复制配置文件 `cp config.yaml.example config.yaml`
3. 编辑配置文件
4. 运行 `./update-server-linux-amd64`

## API

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/check-update?version=v1.0.0` | GET | 检查更新 |
| `/api/v1/version` | GET | 获取最新版本 |
| `/api/v1/webhook` | POST | GitHub Webhook |

## License

MIT
