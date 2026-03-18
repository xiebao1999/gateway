package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Input       InputConfig       `yaml:"input"`
	Cloud       CloudConfig       `yaml:"cloud"`
	Aggregation AggregationConfig `yaml:"aggregation"`
	Logging     LoggingConfig     `yaml:"logging"`
}

type InputConfig struct {
	Host  string `yaml:"host"`
	Port  int    `yaml:"port"`
	Topic string `yaml:"topic"`
}

type CloudConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	ClientID        string `yaml:"client_id"`
	TopicPrefix     string `yaml:"topic_prefix"`
	TopicSeparator  string `yaml:"topic_separator"`  // 分隔符，如 "/" 或 "_"
	DeviceSeparator string `yaml:"device_separator"`  // 设备名与属性分隔符，如 "_" 或 "/"
}

type AggregationConfig struct {
	Mode          string `yaml:"mode"`
	TimerInterval int    `yaml:"timer_interval"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.Input.Host == "" {
		cfg.Input.Host = "0.0.0.0"
	}
	if cfg.Input.Port == 0 {
		cfg.Input.Port = 1883
	}
	if cfg.Cloud.Port == 0 {
		cfg.Cloud.Port = 1883
	}
	if cfg.Cloud.TopicSeparator == "" {
		cfg.Cloud.TopicSeparator = "/"  // 默认用 / 分隔
	}
	if cfg.Cloud.DeviceSeparator == "" {
		cfg.Cloud.DeviceSeparator = "_"  // 默认用 _ 分隔
	}
	if cfg.Aggregation.TimerInterval == 0 {
		cfg.Aggregation.TimerInterval = 5
	}

	return &cfg, nil
}
