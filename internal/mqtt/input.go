package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"mqtt-gateway/config"

	"github.com/eclipse/paho.mqtt.golang"
)

// DeviceDataJSON 设备上报的数据格式
type DeviceDataJSON struct {
	PNs  map[string]string        `json:"PNs"`  // 属性名称映射
	PVs  map[string]interface{}   `json:"PVs"` // 默认值
	Objs []ObjItem                `json:"Objs"` // 变量对象数组
}

type ObjItem struct {
	N  string `json:"N"` // 变量名称（工段_设备_变量属性）
	V1 interface{} `json:"1"` // 实时值
	V2 interface{} `json:"2"` // 时间戳
	V3 interface{} `json:"3"` // 质量戳
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	Segment  string // 工段缩写
	Device   string // 设备名
	Variable string // 属性缩写
	Value    interface{}
	Timestamp interface{}
	Quality   interface{}
}

// InputHandler 处理接收到的设备消息
type InputHandler interface {
	OnMessage(info *DeviceInfo)
}

// InputClient 内置 MQTT Broker 客户端（通过子进程管理 mosquitto）
type InputClient struct {
	client    mqtt.Client
	config    *config.InputConfig
	handler   InputHandler
	mu        sync.Mutex
	connected bool
	mosquitto *exec.Cmd
	stopCh    chan struct{}
}

// NewInputClient 创建输入 MQTT 客户端
func NewInputClient(cfg *config.InputConfig, handler InputHandler) *InputClient {
	c := &InputClient{
		config:  cfg,
		handler: handler,
		stopCh:  make(chan struct{}),
	}

	return c
}

func (c *InputClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
	var data DeviceDataJSON
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		log.Printf("解析 JSON 消息失败: %v", err)
		return
	}

	// 遍历 Objs 处理每个变量
	for _, obj := range data.Objs {
		if obj.N == "" {
			continue
		}

		// 解析变量名：工段_设备_变量属性
		segment, device, attr := ParseVariableName(obj.N)

		// 获取值、时间戳、质量戳
		value := obj.V1
		timestamp := obj.V2
		quality := obj.V3

		// 如果值为 nil，使用 PVs 中的默认值
		if value == nil {
			if defaultVal, ok := data.PVs["1"]; ok {
				value = defaultVal
			}
		}
		if timestamp == nil {
			if defaultVal, ok := data.PVs["2"]; ok {
				timestamp = defaultVal
			}
		}
		if quality == nil {
			if defaultVal, ok := data.PVs["3"]; ok {
				quality = defaultVal
			}
		}

		log.Printf("收到消息 - 工段: %s, 设备: %s, 属性: %s, value: %v",
			segment, device, attr, value)

		// 发送完整信息
		info := &DeviceInfo{
			Segment:  segment,
			Device:   device,
			Variable: attr,
			Value:    value,
			Timestamp: timestamp,
			Quality:  quality,
		}
		c.handler.OnMessage(info)
	}
}

// Start 连接 mosquitto，如果未运行则启动
func (c *InputClient) Start() error {
	addr := fmt.Sprintf("tcp://%s:%d", c.config.Host, c.config.Port)

	// 先尝试直接连接
	if err := c.connectToMosquitto(addr); err == nil {
		return nil
	}

	// 连接失败，尝试启动 mosquitto
	log.Printf("mosquitto 未运行，尝试启动...")
	if err := c.startMosquitto(); err != nil {
		log.Printf("启动 mosquitto 失败: %v", err)
	}

	// 等待 mosquitto 启动
	time.Sleep(2 * time.Second)

	// 再次尝试连接
	return c.connectToMosquitto(addr)
}

// connectToMosquitto 连接 mosquitto
func (c *InputClient) connectToMosquitto(addr string) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(addr)
	opts.SetClientID("gateway-input-" + fmt.Sprintf("%d", time.Now().Unix()))
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetConnectTimeout(5 * time.Second)

	opts.OnConnect = func(client mqtt.Client) {
		c.mu.Lock()
		c.connected = true
		c.mu.Unlock()

		log.Printf("内置 MQTT Broker 已连接: %s", addr)

		if token := client.Subscribe(c.config.Topic, 0, c.messageHandler); token.Wait() && token.Error() != nil {
			log.Printf("订阅 topic 失败: %v", token.Error())
		} else {
			log.Printf("已订阅 topic: %s", c.config.Topic)
		}
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		log.Printf("内置 MQTT Broker 连接丢失: %v", err)
	}

	c.client = mqtt.NewClient(opts)

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("连接内置 MQTT Broker 失败: %w", token.Error())
	}

	return nil
}

// startMosquitto 尝试启动 mosquitto broker
func (c *InputClient) startMosquitto() error {
	// 创建临时配置文件，允许外部访问
	configContent := fmt.Sprintf(`
# 监听所有网络接口，允许外部访问
listener %d 0.0.0.0
allow_anonymous true
persistence false
`, c.config.Port)

	configFile, err := os.CreateTemp("", "mosquitto-*.conf")
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(configContent); err != nil {
		configFile.Close()
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	configFile.Close()

	cmd := exec.Command("mosquitto", "-c", configFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	c.mosquitto = cmd
	log.Printf("mosquitto 进程已启动，PID: %d", cmd.Process.Pid)
	return nil
}

// Stop 停止连接和 mosquitto
func (c *InputClient) Stop() {
	close(c.stopCh)

	if c.client != nil {
		c.client.Disconnect(200)
	}

	if c.mosquitto != nil && c.mosquitto.Process != nil {
		c.mosquitto.Process.Kill()
		c.mosquitto.Wait()
	}
}

// IsConnected 检查是否已连接
func (c *InputClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
