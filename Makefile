# MCP 项目 Makefile

# 变量定义
BINARY_NAME=cowork-database
BUILD_DIR=$(HOME)/go/bin/cowork-database
INSTALL_DIR_USER=$(HOME)/.local/bin
INSTALL_DIR_SYSTEM=/usr/local/bin
GO=go
GOFLAGS=-ldflags="-s -w"

# 获取当前系统信息
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

# 默认目标
.PHONY: all
all: build

# 构建
.PHONY: build
build:
	@echo "构建 $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cowork-database
	@echo "✓ 构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 开发构建（带调试信息）
.PHONY: build-dev
build-dev:
	@echo "构建开发版本..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/cowork-database
	@echo "✓ 开发版本构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 安装到用户目录
.PHONY: install
install: build
	@echo "安装到用户目录..."
	@echo "检查并结束现有的 mcp 服务..."
	@$(MAKE) stop-services
	@mkdir -p $(INSTALL_DIR_USER)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR_USER)/
	@echo "✓ 已安装到: $(INSTALL_DIR_USER)/$(BINARY_NAME)"
	@echo "  请确保 $(INSTALL_DIR_USER) 在您的 PATH 中"

# 安装到系统目录（需要 sudo）
.PHONY: install-system
install-system: build
	@echo "安装到系统目录..."
	@echo "检查并结束现有的 mcp 服务..."
	@$(MAKE) stop-services
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR_SYSTEM)/
	@echo "✓ 已安装到: $(INSTALL_DIR_SYSTEM)/$(BINARY_NAME)"

# 停止现有的 mcp 服务
.PHONY: stop-services
stop-services:
	@echo "  查找正在运行的 mcp 进程..."
	@pids="$$(pgrep -f '^.*mcp (mysql|redis|pulsar)' 2>/dev/null || true)"; \
	if [ -n "$$pids" ]; then \
		echo "  发现运行中的 mcp 进程: $$pids"; \
		echo "  正在终止这些进程..."; \
		echo "$$pids" | xargs -r kill -TERM 2>/dev/null || true; \
		sleep 1; \
		remaining="$$(echo "$$pids" | xargs -r ps -p 2>/dev/null | grep -v PID | wc -l || echo 0)"; \
		if [ "$$remaining" -gt 0 ]; then \
			echo "  强制终止剩余进程..."; \
			echo "$$pids" | xargs -r kill -KILL 2>/dev/null || true; \
		fi; \
		echo "  ✓ mcp 服务已停止"; \
	else \
		echo "  ✓ 未发现运行中的 mcp 服务"; \
	fi

# 卸载（用户目录）
.PHONY: uninstall
uninstall:
	@echo "从用户目录卸载..."
	@echo "检查并结束现有的 mcp 服务..."
	@$(MAKE) stop-services
	@rm -f $(INSTALL_DIR_USER)/$(BINARY_NAME)
	@echo "✓ 已卸载"

# 卸载（系统目录）
.PHONY: uninstall-system
uninstall-system:
	@echo "从系统目录卸载..."
	@echo "检查并结束现有的 mcp 服务..."
	@$(MAKE) stop-services
	@sudo rm -f $(INSTALL_DIR_SYSTEM)/$(BINARY_NAME)
	@echo "✓ 已卸载"

# 清理
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -rf dist/
	@rm -f ./mcp
	@echo "✓ 清理完成"

# 运行测试
.PHONY: test
test:
	@echo "运行测试..."
	@$(GO) test -v ./...

# 运行 MCP 测试
.PHONY: test-mcp
test-mcp: build
	@echo "运行 MCP 功能测试..."
	@python3 test/test_mysql_mcp.py

# 交叉编译
.PHONY: build-all
build-all:
	@echo "交叉编译所有平台..."
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/cowork-database
	@GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/cowork-database
	@GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/cowork-database
	@GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/cowork-database
	@GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/cowork-database
	@echo "✓ 交叉编译完成"
	@ls -lh dist/

# 创建发布包
.PHONY: release
release: build-all
	@echo "创建发布包..."
	@mkdir -p release
	@VERSION=$${VERSION:-latest}; \
	cd $(BUILD_DIR) && \
	for file in mcp-*; do \
		echo "打包 $$file..."; \
		if [[ "$$file" == *.exe ]]; then \
			zip ../release/$$file-$$VERSION.zip $$file ../LICENSE ../README.md; \
		else \
			tar -czf ../release/$$file-$$VERSION.tar.gz $$file ../LICENSE ../README.md; \
		fi; \
	done
	@cd release && sha256sum * > checksums.txt
	@echo "✓ 发布包创建完成，文件位于 release/ 目录"
	@ls -lh release/

# 开发模式（构建并运行）
.PHONY: dev
dev: build-dev
	@echo "启动 MySQL MCP 服务器..."
	@$(BUILD_DIR)/$(BINARY_NAME) mysql

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@$(GO) fmt ./...
	@echo "✓ 格式化完成"

# 检查代码
.PHONY: lint
lint:
	@echo "检查代码..."
	@golangci-lint run || echo "提示: 安装 golangci-lint: https://golangci-lint.run/usage/install/"

# 显示帮助
.PHONY: help
help:
	@echo "MCP 项目 Makefile 使用说明:"
	@echo ""
	@echo "构建和安装:"
	@echo "  make build          - 构建项目到 bin/ 目录"
	@echo "  make build-dev      - 构建开发版本（包含调试信息）"
	@echo "  make install        - 安装到 ~/.local/bin"
	@echo "  make install-system - 安装到 /usr/local/bin（需要 sudo）"
	@echo ""
	@echo "开发相关:"
	@echo "  make dev            - 构建并启动 MySQL MCP 服务器"
	@echo "  make test           - 运行单元测试"
	@echo "  make test-mcp       - 运行 MCP 功能测试"
	@echo "  make fmt            - 格式化代码"
	@echo "  make lint           - 代码检查"
	@echo ""
	@echo "清理和卸载:"
	@echo "  make clean          - 清理构建文件"
	@echo "  make stop-services  - 停止现有的 mcp 服务"
	@echo "  make uninstall      - 从用户目录卸载"
	@echo "  make uninstall-system - 从系统目录卸载"
	@echo ""
	@echo "其他:"
	@echo "  make build-all      - 交叉编译所有平台"
	@echo "  make help           - 显示此帮助信息"

.DEFAULT_GOAL := help