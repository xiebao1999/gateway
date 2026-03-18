package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"mqtt-gateway/config"

	"github.com/eclipse/paho.mqtt.golang"
)

// OutputClient 云端 MQTT 客户端（发布数据）
type OutputClient struct {
	client    mqtt.Client
	config    *config.CloudConfig
	mu        sync.Mutex
	connected bool
}

// NewOutputClient 创建输出 MQTT 客户端
func NewOutputClient(cfg *config.CloudConfig) *OutputClient {
	c := &OutputClient{
		config: cfg,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port))
	opts.SetClientID(cfg.ClientID)
	opts.SetAutoReconnect(true)
	opts.SetKeepAlive(30 * time.Second)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		opts.SetPassword(cfg.Password)
	}

	opts.OnConnect = func(client mqtt.Client) {
		c.mu.Lock()
		c.connected = true
		c.mu.Unlock()
		log.Printf("云端 MQTT Broker 已连接: %s:%d", cfg.Host, cfg.Port)
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		log.Printf("云端 MQTT Broker 连接丢失: %v", err)
	}

	c.client = mqtt.NewClient(opts)
	return c
}

// Connect 连接到云端 MQTT Broker
func (c *OutputClient) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("连接云端 MQTT Broker 失败: %w", token.Error())
	}
	return nil
}

// Publish 发布消息
// topicKey 格式：工段_设备 (如 "MZ/YC01")
func (c *OutputClient) Publish(topicKey string, data interface{}) error {
	// 检查连接状态，未连接则尝试重连
	if !c.IsConnected() {
		log.Printf("云端 MQTT 未连接，尝试重连...")
		if err := c.Connect(); err != nil {
			return fmt.Errorf("重连失败: %w", err)
		}
	}

	// 使用配置的分隔符拼接 topic
	topic := fmt.Sprintf("%s%s%s", c.config.TopicPrefix, c.config.TopicSeparator, topicKey)

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if token := c.client.Publish(topic, 1, false, payload); token.Wait() && token.Error() != nil {
		return fmt.Errorf("发布消息失败: %w", token.Error())
	}

	log.Printf("已发布到 topic: %s, payload: %s", topic, string(payload))
	return nil
}

// Disconnect 断开连接
func (c *OutputClient) Disconnect() {
	c.client.Disconnect(200)
}

// IsConnected 检查是否已连接
func (c *OutputClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
