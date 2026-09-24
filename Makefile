# QuestOS (timeWeave) Makefile

APP       := timeweave
BIN_DIR   := bin
MAIN_PKG  := ./cmd/server

# ============== 通用 ==============

.PHONY: help
help:
	@echo "make run       启动服务（加载 .env）"
	@echo "make build     编译二进制到 $(BIN_DIR)/"
	@echo "make test      跑单元测试"
	@echo "make vet       go vet"
	@echo "make tidy      go mod tidy"
	@echo "make fmt       gofmt 格式化"
	@echo "make migrate   在数据库里跑 AutoMigrate（要求服务停掉）"

# ============== 启动 ==============

.PHONY: run
run:
	go run $(MAIN_PKG)

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP) $(MAIN_PKG)
	@echo "built: $(BIN_DIR)/$(APP)"

# ============== 质量 ==============

.PHONY: test
test:
	go test ./... -v

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: tidy
tidy:
	go mod tidy

# ============== 工具 ==============

# 清理本地 MySQL 的 questos 库（危险，慎用）
.PHONY: db-reset
db-reset:
	mysql -u root -p -e "DROP DATABASE IF EXISTS questos; CREATE DATABASE questos DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
	@echo "database reset"

# 查看依赖树
.PHONY: deps
deps:
	go list -m all