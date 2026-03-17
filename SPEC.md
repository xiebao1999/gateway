# MQTT 数据网关 - 需求规格说明书

## 1. 项目概述

- **项目名称**: MQTT 数据网关 (MQTT Bridge Gateway)
- **项目类型**: MQTT 消息汇聚与转发服务
- **核心功能**: 接收设备上报的 MQTT 消息，按设备汇聚数据，转换为拼音首字母后打包发布到云端
- **目标用户**: 工业物联网数据采集场景

## 2. 功能需求

### 2.1 内置 MQTT Broker

- 内置轻量级 MQTT Broker（通过子进程管理 mosquitto），用于接收设备数据
- 支持配置监听地址和端口
- 支持配置订阅的 topic

### 2.2 消息接收与解析

#### 2.2.1 输入消息格式

设备通过 MQTT 上报 JSON 数据，格式如下：

```json
{
    "PNs": {
        "1": "V",
        "2": "T",
        "3": "Q"
    },
    "PVs": {
        "1": 0,
        "2": "2019-03-01 08:00:00.000",
        "3": 192
    },
    "Objs": [
        {
            "N": "工段_设备_变量属性",
            "1": 值,
            "2": 时间戳,
            "3": 质量戳
        }
    ]
}
```

- `PNs`: 变量属性名称映射（1=值, 2=时间戳, 3=质量戳）
- `PVs`: 变量属性默认值数组
- `Objs`: 变量对象数组
- `N`: 变量名称，格式为 `工段_设备_变量属性`
- `1`, `2`, `3`: 分别对应值、时间戳、质量戳

### 2.3 中文转拼音首字母

- 变量名（`Objs.N`）按 `_` 切割为三部分
- 工段 → 拼音首字母缩写（如 "磨重" → "MZ"）
- 设备 → 保留数字 + 拼音首字母（如 "摇床01" → "YC01"）
- 属性 → 拼音首字母缩写（如 "水流量" → "SLL"）

**转换示例**：
| 输入 | 工段 | 设备 | 属性 |
|------|------|------|------|
| 磨重_摇床01_水流量 | MZ | YC01 | SLL |
| 发酵_罐02_温度 | FJ | G02 | WD |
| 提取_离心机03_转速 | TQ | LXJ03 | ZS |

### 2.4 数据汇聚

- 按 `工段_设备` 汇聚数据
- 汇聚后的 JSON 格式：
```json
{
    "reported": {
        "属性1": 值1,
        "属性2": 值2
    },
    "time": "2025-08-10 17:21:03",
    "device_id": "YC01"
}
```

### 2.5 数据发布

- 输出 topic 格式: `gateway/{工段缩写}/{设备缩写}`（如 `gateway/MZ/YC01`）
- 发布模式（可配置）：
  - **实时模式 (realtime)**: 收到单条消息立即发布
  - **定时模式 (timer)**: 定时批量发布，默认 5 秒

### 2.6 云端 MQTT 连接

- 可配置云端 Broker 地址、端口
- 可配置用户名、密码
- 可配置 Client ID
- 可配置输出 topic 前缀（默认 "gateway"）

## 3. 配置需求

配置文件 `config.yaml`：

```yaml
# 内置 MQTT Broker 配置（接收设备数据）
input:
  host: "0.0.0.0"
  port: 1883
  topic: "datachange_S_KIO_Project"  # 设备上报的 topic

# 云端 MQTT Broker 配置（输出数据）
cloud:
  host: "106.13.190.210"      # 云端 Broker 地址
  port: 1883
  username: ""                 # 用户名（可选）
  password: ""                 # 密码（可选）
  client_id: "mqtt-gateway"  # Client ID
  topic_prefix: "gateway"    # 输出 topic 前缀

# 汇聚配置
aggregation:
  mode: "timer"              # realtime(实时) 或 timer(定时)
  timer_interval: 5          # 定时发布间隔（秒）

# 日志配置
logging:
  level: "info"              # debug, info, warn, error
```

## 4. 数据流转

```
设备
  │
  │ MQTT: datachange_S_KIO_Project
  │ {"Objs":[{"N":"磨重_摇床01_水流量","1":25.5,"2":...,"3":...}]}
  ▼
┌─────────────────────────────────────────┐
│  1. 解析 JSON                            │
│  2. 提取 Objs.N                         │
│  3. 切割: 磨重_摇床01_水流量            │
│     - 工段: 磨重 → MZ                   │
│     - 设备: 摇床01 → YC01               │
│     - 属性: 水流量 → SLL                │
│  4. 按 工段_设备 汇聚                   │
└─────────────────────────────────────────┘
  │
  │ 定时/实时触发
  ▼
┌─────────────────────────────────────────┐
│  发布到云端                              │
│  Topic: gateway/MZ/YC01                │
│  Payload: {"reported":{"SLL":25.5},     │
│            "time":"2026-03-17 20:36:32",│
│            "device_id":"YC01"}          │
└─────────────────────────────────────────┘
  │
  ▼
云端 MQTT Broker
```

## 5. 部署需求

### 5.1 系统要求

- Linux 操作系统（x86_64 或 ARM）
- Go 1.21+

### 5.2 依赖

- **mosquitto**: 内置 MQTT Broker（需要安装）
- **Go 库**:
  - `github.com/eclipse/paho.mqtt.golang` - MQTT 客户端
  - `github.com/mozillazg/go-pinyin` - 中文转拼音
  - `gopkg.in/yaml.v3` - YAML 配置解析

### 5.3 编译

```bash
# 当前平台
make build

# ARM64 (开发板)
make build-arm64

# ARMv7
make build-arm
```

## 6. 目录结构

```
gateway/
├── config/
│   └── config.go           # 配置加载
├── internal/
│   ├── mqtt/
│   │   ├── input.go       # 内置 MQTT 客户端
│   │   ├── output.go      # 云端 MQTT 客户端
│   │   └── pinyin.go      # 中文转拼音首字母
│   └── aggregator/
│       └── aggregator.go  # 数据汇聚逻辑
├── config.yaml             # 配置文件
├── go.mod                  # Go 模块
├── Makefile               # 编译脚本
├── main.go                # 主程序
├── SPEC.md                # 需求文档
└── DEPLOY.md              # 部署文档
```

## 7. 验收标准

1. ✅ 服务可正常启动，内置 mosquitto 运行成功
2. ✅ 设备上报消息能正确解析
3. ✅ 中文变量名能正确转换为拼音首字母
4. ✅ 实时模式下，收到消息立即转发到云端
5. ✅ 定时模式下，按配置间隔转发数据到云端
6. ✅ 输出 topic 格式正确: `gateway/{工段缩写}/{设备缩写}`
7. ✅ 输出 JSON 格式正确: `{"reported":{},"time":"","device_id":""}`
8. ✅ 可通过配置文件修改所有参数
9. ✅ 支持交叉编译为 ARM 架构可执行文件
