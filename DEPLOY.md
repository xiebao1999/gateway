# MQTT 数据网关 - 部署文档

## 系统要求

- Linux 操作系统（x86_64 或 ARM）
- Go 1.21+

## 依赖安装

### 1. 安装 mosquitto（内置 MQTT Broker）

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install -y mosquitto mosquitto-clients
```

**CentOS/RHEL:**
```bash
sudo yum install -y mosquitto mosquitto-clients
```

**启动 mosquitto:**
```bash
# 允许匿名访问（开发环境）
sudo mosquitto -p 1883 -c /dev/null

# 或创建配置文件 /etc/mosquitto/conf.d/allow匿名.conf
# listen 1883
# allow_anonymous true
```

### 2. 安装 Go（编译用）

**Ubuntu/Debian:**
```bash
sudo apt-get install -y golang-go
```

**从官网下载:**
```bash
curl -fsSL https://go.dev/dl/go1.21.6.linux-amd64.tar.gz -o /tmp/go.tar.gz
sudo tar -C /usr/local -xzf /tmp/go.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

## 编译

### 1. 克隆/进入项目目录

```bash
cd gateway
```

### 2. 下载依赖

```bash
go mod tidy
```

### 3. 编译

```bash
# 当前平台
make build

# Linux x86_64
make build-linux

# ARM64 (推荐开发板)
make build-arm64

# ARMv7
make build-arm
```

编译产物位于 `bin/` 目录：
- `mqtt-gateway` - 当前平台
- `mqtt-gateway-linux-amd64` - x86_64
- `mqtt-gateway-linux-arm64` - ARM64
- `mqtt-gateway-linux-armv7` - ARMv7

## 配置

编辑 `config.yaml`：

```yaml
# 内置 MQTT Broker 配置（接收设备数据）
input:
  host: "0.0.0.0"
  port: 1883
  topic: "device/data"

# 云端 MQTT Broker 配置（输出数据）
cloud:
  host: "your-cloud-broker.com"  # 修改为云端 Broker 地址
  port: 1883
  username: ""                    # 云端用户名（如需要）
  password: ""                    # 云端密码（如需要）
  client_id: "mqtt-gateway"
  topic_prefix: "gateway"

# 汇聚配置
aggregation:
  mode: "timer"        # realtime(实时) 或 timer(定时)
  timer_interval: 5    # 定时发布间隔（秒）

# 消息字段映射
message:
  device_id_field: "device_id"
  variable_field: "variable"
  value_field: "value"
```

## 运行

```bash
# 前台运行
./bin/mqtt-gateway -config config.yaml

# 后台运行
nohup ./bin/mqtt-gateway -config config.yaml > gateway.log 2>&1 &
```

## 验证

### 1. 测试设备上报

使用 mosquitto_pub 模拟设备上报数据：

```bash
mosquitto_pub -t device/data -m '{
  "device_id": "AM2025040100609",
  "variable": "factory1_device1_temperature",
  "value": 25.5
}'
```

### 2. 验证云端接收

查看日志确认消息已发布到云端：

```bash
tail -f gateway.log
```

或使用 mosquitto_sub 订阅云端 topic：

```bash
mosquitto_sub -t "gateway/AM2025040100609"
```

## 服务管理

### 使用 systemd（推荐）

创建服务文件 `/etc/systemd/system/mqtt-gateway.service`:

```ini
[Unit]
Description=MQTT Gateway
After=network.target mosquitto.service

[Service]
Type=simple
User=root
WorkingDirectory=/path/to/gateway
ExecStart=/path/to/gateway/bin/mqtt-gateway -config config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable mqtt-gateway
sudo systemctl start mqtt-gateway

# 查看状态
sudo systemctl status mqtt-gateway

# 查看日志
sudo journalctl -u mqtt-gateway -f
```

## 常见问题

### 1. mosquitto 连接被拒绝

确保 mosquitto 已启动：
```bash
sudo systemctl start mosquitto
sudo systemctl status mosquitto
```

### 2. 云端连接失败

- 检查云端 Broker 地址和端口
- 检查用户名密码是否正确
- 检查网络连通性

### 3. 消息格式错误

确保设备上报的 JSON 包含必需字段：
- `device_id` - 设备ID
- `variable` - 变量名（工段_设备_变量属性格式）
- `value` - 变量值
