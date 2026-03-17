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
