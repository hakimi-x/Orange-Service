.PHONY: build run test clean dev

# 应用名称
APP_NAME := update-server
# 构建目录
BUILD_DIR := bin

# 构建
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# 运行
run: build
	CONFIG_PATH=configs/config.yaml $(BUILD_DIR)/$(APP_NAME)

# 开发模式 (热重载)
dev:
	CONFIG_PATH=configs/config.yaml air

# 测试
test:
	go test -v ./...

# 测试覆盖率
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 清理
clean:
	rm -rf $(BUILD_DIR) tmp coverage.out coverage.html

# 格式化
fmt:
	go fmt ./...

# 检查
lint:
	golangci-lint run

# 安装依赖
deps:
	go mod tidy
	go mod download

# 生成 Swagger 文档
swagger:
	swag init

# 安装开发工具
tools:
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/air-verse/air@latest
