package aggregator

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"mqtt-gateway/config"
	"mqtt-gateway/internal/mqtt"
)

// DeviceData 设备数据结构
type DeviceData struct {
	Reported map[string]interface{} `json:"reported"`
	Time     string                 `json:"time"`
	DeviceID string                 `json:"device_id"`
}

// Aggregator 数据汇聚器
type Aggregator struct {
	config    *config.AggregationConfig
	output    *mqtt.OutputClient
	data      map[string]map[string]interface{} // key = "segment_device", value = variable->value
	mu        sync.RWMutex
	stopCh    chan struct{}
}

// New 创建数据汇聚器
func New(cfg *config.AggregationConfig, output *mqtt.OutputClient) *Aggregator {
	return &Aggregator{
		config: cfg,
		output: output,
		data:   make(map[string]map[string]interface{}),
		stopCh: make(chan struct{}),
	}
}

// OnMessage 处理接收到的消息（实现 mqtt.InputHandler 接口）
func (a *Aggregator) OnMessage(info *mqtt.DeviceInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// key = 工段_设备
	key := info.Segment + "_" + info.Device

	// 初始化设备数据
	if _, ok := a.data[key]; !ok {
		a.data[key] = make(map[string]interface{})
	}

	// 更新变量值
	a.data[key][info.Variable] = info.Value

	// 实时模式：立即发布
	if a.config.Mode == "realtime" {
		a.publishDevice(key, info.Segment, info.Device)
	}
}

// Start 启动定时器（仅在定时模式下有效）
func (a *Aggregator) Start() {
	if a.config.Mode != "timer" {
		log.Printf("聚合模式为实时模式，不启动定时器")
		return
	}

	interval := time.Duration(a.config.TimerInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("启动定时发布模式，间隔: %v", interval)

	for {
		select {
		case <-ticker.C:
			a.publishAll()
		case <-a.stopCh:
			log.Println("定时器已停止")
			return
		}
	}
}

// Stop 停止定时器
func (a *Aggregator) Stop() {
	close(a.stopCh)
}

// publishDevice 发布单个设备的数据
func (a *Aggregator) publishDevice(key, segment, deviceID string) {
	a.mu.RLock()
	variables, ok := a.data[key]
	a.mu.RUnlock()

	if !ok || len(variables) == 0 {
		return
	}

	// 创建输出数据
	outputData := DeviceData{
		Reported: make(map[string]interface{}),
		Time:     time.Now().Format("2006-01-02 15:04:05"),
		DeviceID: deviceID,
	}

	for k, v := range variables {
		outputData.Reported[k] = v
	}

	// 发布到 topic: gateway/{工段}/{设备}
	topicKey := segment + "/" + deviceID
	if err := a.output.Publish(topicKey, outputData); err != nil {
		log.Printf("发布设备 %s 数据失败: %v", topicKey, err)
	}

	// 发布后清除已发布的数据
	a.mu.Lock()
	delete(a.data, key)
	a.mu.Unlock()
}

// publishAll 发布所有设备的数据
func (a *Aggregator) publishAll() {
	a.mu.RLock()
	keys := make([]string, 0, len(a.data))
	for k := range a.data {
		keys = append(keys, k)
	}
	a.mu.RUnlock()

	if len(keys) == 0 {
		log.Println("无数据需要发布")
		return
	}

	log.Printf("定时发布 %d 个设备的数据", len(keys))

	for _, key := range keys {
		parts := splitKey(key)
		if len(parts) == 2 {
			a.publishDevice(key, parts[0], parts[1])
		}
	}
}

// splitKey 分割 key 为 segment 和 device
func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == '_' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key, key}
}

// GetData 获取当前所有设备数据（用于调试）
func (a *Aggregator) GetData() map[string]map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for k, v := range a.data {
		result[k] = make(map[string]interface{})
		for k2, v2 := range v {
			result[k][k2] = v2
		}
	}
	return result
}

// ToJSON 将数据转为 JSON 字符串（用于调试）
func (a *Aggregator) ToJSON() string {
	data := a.GetData()
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
