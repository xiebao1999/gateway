# MQTT 数据网关 - 部署文档

## 一、编译 ARM 版本

### 1.1 在本地电脑编译

```bash
cd /home/xiebaolai/claude/gateway

# 编译 ARM64 版本（推荐）
make build-arm64

# 或编译 ARMv7 版本（32位）
make build-arm
```

编译产物位于 `bin/` 目录：
- `mqtt-gateway-linux-arm64` - ARM64 (AArch64)
- `mqtt-gateway-linux-armv7` - ARMv7 (32位)

---

## 二、传输文件到开发板

### 2.1 方法一：SCP

```bash
# 传输可执行文件
scp bin/mqtt-gateway-linux-arm64 user@192.168.1.100:/root/gateway/

# 传输配置文件
scp config.yaml user@192.168.1.100:/root/gateway/
```

### 2.2 方法二：SFTP

使用 FileZilla 或其他 SFTP 客户端连接开发板传输文件。

---

## 三、开发板环境准备

### 3.1 确认架构

```bash
# 查看 CPU 架构
uname -a
```

### 3.2 安装 mosquitto

**Ubuntu/Debian (ARM):**
```bash
sudo apt-get update
sudo apt-get install -y mosquitto mosquitto-clients
```

### 3.3 配置 mosquitto 允许外部访问

创建配置文件 `/etc/mosquitto/conf.d/listener.conf`:

```bash
# 监听所有网络接口，允许外部访问
listener 1883 0.0.0.0
allow_anonymous true
```

重启 mosquitto：

```bash
sudo systemctl restart mosquitto
```

### 3.4 设置 mosquitto 开机自启

```bash
sudo systemctl enable mosquitto
sudo systemctl start mosquitto
```

### 3.5 开放防火墙端口

```bash
sudo ufw allow 1883/tcp
sudo ufw reload
```

### 3.6 添加执行权限

```bash
chmod +x /root/gateway/mqtt-gateway-linux-arm64
```

### 3.7 WiFi 连接手机热点

#### 3.7.1 开启 WiFi

```bash
# 开启 WiFi
sudo nmcli radio wifi on
```

#### 3.7.2 扫描并连接热点

```bash
# 扫描可用热点
sudo nmcli device wifi list

# 连接手机热点（替换为热点名称和密码）
sudo nmcli device wifi connect "xiebaolai" password "11111111"
```

#### 3.7.3 设置 WiFi 开机自动连接

连接成功后，执行：

```bash
# 查看已连接的热点名称
nmcli -t -f NAME connection show

# 设置自动连接（替换热点名称）
sudo nmcli connection modify "xiebaolai" connection.autoconnect yes
```

或者通过 rc.local 开机启动：

```bash
sudo vi /etc/rc.local
```

在 `exit 0` 前添加：

```bash
# 等待 WiFi 启动
sleep 10
sudo nmcli device wifi connect "xiebaolai" password "11111111"
sleep 5
sudo ntpdate -b pool.ntp.org
```

#### 3.7.4 查看 WiFi 状态

```bash
# 查看 IP 地址
ip addr show wlan0

# 测试网络连接
ping -c 3 8.8.8.8
```

### 3.8 设置时间同步（防止重启后时间丢失）

```bash
# 安装时间同步工具
sudo apt-get update
sudo apt-get install -y ntpdate

# 设置时区
sudo timedatectl set-timezone Asia/Shanghai

# 手动同步一次时间
sudo ntpdate -b pool.ntp.org
```

**设置开机自动同步时间**：在 rc.local 中添加同步命令：

```bash
sudo vi /etc/rc.local
```

在 `exit 0` 前添加：

```bash
sudo ntpdate -b pool.ntp.org
```

---

## 四、运行服务

### 4.1 后台运行（推荐）

```bash
cd /root/gateway

# 创建日志目录
mkdir -p /root/gateway/logs

# 后台运行，日志保存到 logs/gateway.log
nohup ./mqtt-gateway-linux-arm64 -config config.yaml > /root/gateway/logs/gateway.log 2>&1 &

# 查看是否运行成功
ps aux | grep mqtt-gateway
```

### 4.2 开机自启

编辑 `/etc/rc.local`，在 `exit 0` 前添加：

```bash
# 等待 WiFi 和时间同步
sleep 15
sudo ntpdate -b pool.ntp.org
sleep 2

# 启动 MQTT 网关
cd /root/gateway
nohup ./mqtt-gateway-linux-arm64 -config config.yaml > /root/gateway/logs/gateway.log 2>&1 &
```

赋予执行权限：
```bash
sudo chmod +x /etc/rc.local
```

---

## 五、日志管理

### 5.1 查看日志

```bash
# 实时查看
tail -f /root/gateway/logs/gateway.log

# 查看最后100行
tail -n 100 /root/gateway/logs/gateway.log
```

### 5.2 配置日志自动轮转

创建 `/etc/logrotate.d/mqtt-gateway`:

```
/root/gateway/logs/gateway.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    create 0644 root root
}
```

### 5.3 手动轮转日志

```bash
sudo logrotate -f /etc/logrotate.d/mqtt-gateway
```

---

## 六、验证

### 6.1 检查进程

```bash
ps aux | grep mqtt-gateway
```

### 6.2 验证 mosquitto 监听

```bash
netstat -tlnp | grep 1883
```

### 6.3 测试外部设备访问

在其他电脑上测试连接开发板：

```bash
# 发布测试
mosquitto_pub -h 192.168.1.100 -t datachange_S_KIO_Project -m '{
  "PNs": {"1": "V"},
  "Objs": [
    {"N": "磨重_摇床01_水流量", "1": 25.5}
  ]
}'
```

### 6.4 查看日志输出

```bash
tail -f /root/gateway/logs/gateway.log
```

---

## 七、目录结构

部署后的目录结构：

```
/root/gateway/
├── mqtt-gateway-linux-arm64    # 主程序
├── config.yaml                  # 配置文件
└── logs/
    ├── gateway.log             # 当前日志
    ├── gateway.log.1           # 轮转后的日志
    └── gateway.log.2.gz        # 压缩的旧日志
```

---

## 八、常用命令

| 操作 | 命令 |
|------|------|
| 启动服务 | `cd /root/gateway && nohup ./mqtt-gateway-linux-arm64 -config config.yaml > logs/gateway.log 2>&1 &` |
| 停止服务 | `pkill mqtt-gateway` |
| 查看日志 | `tail -f /root/gateway/logs/gateway.log` |
| 重启服务 | `pkill mqtt-gateway && cd /root/gateway && nohup ./mqtt-gateway-linux-arm64 -config config.yaml > logs/gateway.log 2>&1 &` |
| 手动轮转日志 | `sudo logrotate -f /etc/logrotate.d/mqtt-gateway` |
