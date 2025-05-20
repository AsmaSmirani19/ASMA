package agent

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sender struct {
		ID   string `yaml:"id"`
		IP   string `yaml:"ip"`
		Port int    `yaml:"port"`
	} `yaml:"sender"`

	Reflector struct {
		IP   string `yaml:"ip"`
		Port int    `yaml:"port"`
	} `yaml:"reflector"`

	GRPC struct {
		Port string `yaml:"port"`
	} `yaml:"grpc"`

	Server struct {
		Main string `yaml:"main"`
	} `yaml:"server"`

	Network struct {
		ListenPort int           `yaml:"listen_port"`
		PacketSize int           `yaml:"packet_size"`
		Timeout    time.Duration `yaml:"timeout"`
	} `yaml:"network"`

	DefaultTest struct {
		Duration   time.Duration `yaml:"duration"`
		Interval   time.Duration `yaml:"interval"`
		TargetIP   string        `yaml:"target_ip"`
		TargetPort int           `yaml:"target_port"`
	} `yaml:"default_test"`

	Kafka struct {
		Brokers           []string `yaml:"brokers"`
		TestRequestTopic  string   `yaml:"test_request_topic"`
		TestResultTopic   string   `yaml:"test_result_topic"`
		GroupID           string   `yaml:"group_id"`
	} `yaml:"kafka"`

	WebSocket struct {
		URL string `yaml:"url"`
	} `yaml:"websocket"` 
}

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
