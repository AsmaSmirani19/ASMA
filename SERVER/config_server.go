package server

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)


var AppConfig Config

func LoadConfig(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Erreur lecture config: %v", err)
	}

	if err := yaml.Unmarshal(data, &AppConfig); err != nil {
		log.Fatalf("Erreur parsing config YAML: %v", err)
	}
}

type Config struct {
	Server struct {
		TCPListener struct {
			Address  string `yaml:"address"`
			Protocol string `yaml:"protocol"`
		} `yaml:"tcp_listener"`
	} `yaml:"server"`

	Network struct {
		ServerAddress string `yaml:"server_address"`
		ServerPort    int    `yaml:"server_port"`
		SenderPort    int    `yaml:"sender_port"`
		ReceiverPort  int    `yaml:"receiver_port"`
		Timeout       int    `yaml:"timeout"`  
	} `yaml:"network"`

	QuickTest struct {
		TestID     string `yaml:"test_id"`
		Parameters string `yaml:"parameters"`
	} `yaml:"quick_test"`

		GRPC struct {
			Address string `yaml:"address"`
			Port    int    `yaml:"port"`
		} `yaml:"grpc"`
	
	WebSocket struct {
		Address string `yaml:"address"`
		Port    int    `yaml:"port"`
	} `yaml:"websocket"`

	Kafka struct {
		Brokers          []string `yaml:"brokers"`
		TestRequestTopic string   `yaml:"test_request_topic"`
	} `yaml:"kafka"`
	
}
	



