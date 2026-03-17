.PHONY: all build build-linux build-arm build-arm64 clean test

# 输出目录
OUTPUT_DIR = bin

# 默认目标：编译当前平台
all: build

# 编译当前平台
build:
	@echo "编译当前平台..."
	@mkdir -p $(OUTPUT_DIR)
	@go build -o $(OUTPUT_DIR)/mqtt-gateway .

# 编译 Linux x86_64
build-linux:
	@echo "编译 Linux x86_64..."
	@mkdir -p $(OUTPUT_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(OUTPUT_DIR)/mqtt-gateway-linux-amd64 .

# 编译 Linux ARM v7 (ARMv7)
build-arm:
	@echo "编译 Linux ARM v7..."
	@mkdir -p $(OUTPUT_DIR)
	@GOOS=linux GOARCH=arm GOARM=7 go build -o $(OUTPUT_DIR)/mqtt-gateway-linux-armv7 .

# 编译 Linux ARM64 (ARMv8)
build-arm64:
	@echo "编译 Linux ARM64..."
	@mkdir -p $(OUTPUT_DIR)
	@GOOS=linux GOARCH=arm64 go build -o $(OUTPUT_DIR)/mqtt-gateway-linux-arm64 .

# 编译所有平台
build-all: build-linux build-arm build-arm64
	@echo "所有平台编译完成!"
	@ls -lh $(OUTPUT_DIR)

# 清理
clean:
	@rm -rf $(OUTPUT_DIR)

# 运行
run:
	@go run main.go -config config.yaml

# 测试
test:
	@go test -v ./...

# 下载依赖
deps:
	@go mod tidy
	@go mod download
