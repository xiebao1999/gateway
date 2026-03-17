package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mqtt-gateway/config"
	"mqtt-gateway/internal/aggregator"
	"mqtt-gateway/internal/mqtt"
)

var (
	configPath = flag.String("config", "config.yaml", "配置文件路径")
	version    = flag.Bool("version", false, "显示版本信息")
)

const AppVersion = "1.0.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("MQTT Gateway v%s\n", AppVersion)
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("MQTT Gateway v%s 启动中...", AppVersion)
	log.Printf("配置: 内置 Broker %s:%d, Topic: %s", cfg.Input.Host, cfg.Input.Port, cfg.Input.Topic)
	log.Printf("配置: 云端 Broker %s:%d, TopicPrefix: %s", cfg.Cloud.Host, cfg.Cloud.Port, cfg.Cloud.TopicPrefix)
	log.Printf("配置: 聚合模式: %s, 间隔: %ds", cfg.Aggregation.Mode, cfg.Aggregation.TimerInterval)

	// 创建云端 MQTT 客户端
	outputClient := mqtt.NewOutputClient(&cfg.Cloud)

	// 连接云端 MQTT Broker
	log.Println("正在连接云端 MQTT Broker...")
	if err := outputClient.Connect(); err != nil {
		log.Printf("警告: 连接云端 MQTT Broker 失败: %v", err)
		log.Println("将继续尝试重连...")
	} else {
		log.Println("云端 MQTT Broker 连接成功")
	}

	// 等待云端连接成功
	for i := 0; i < 30; i++ {
		if outputClient.IsConnected() {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 创建数据汇聚器
	agg := aggregator.New(&cfg.Aggregation, outputClient)

	// 创建内置 MQTT 客户端
	inputClient := mqtt.NewInputClient(&cfg.Input, agg)

	// 启动内置 MQTT（会尝试启动 mosquitto）
	log.Println("正在启动内置 MQTT Broker...")
	if err := inputClient.Start(); err != nil {
		log.Printf("警告: 启动内置 MQTT 失败: %v", err)
		log.Println("请确保 mosquitto 已安装: sudo apt-get install mosquitto")
	}

	// 启动定时发布（仅定时模式）
	if cfg.Aggregation.Mode == "timer" {
		go agg.Start()
	}

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("收到退出信号，正在关闭...")

	// 停止定时器
	agg.Stop()

	// 断开云端连接
	outputClient.Disconnect()

	// 停止内置 MQTT
	inputClient.Stop()

	log.Println("服务已关闭")
}
