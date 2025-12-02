package pkg

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/FerroO2000/goccia"
	"github.com/FerroO2000/goccia/egress"
	"github.com/FerroO2000/goccia/ingress"
	"github.com/FerroO2000/goccia/processor"
	"github.com/caarlos0/env"
	"github.com/goccy/go-yaml"
	"github.com/squadracorsepolito/acmelib"
)

const defaultConfigPath = "/app/config/config.yaml"

func defaultConfig() *Config {
	return &Config{
		ServiceName:   "sc-telemetry",
		ConnectorSize: 2048,
		UDP: &UDPStageConfig{
			IPAddr: "127.0.0.1",
			Port:   20_000,
		},
		Cannelloni: &CannelloniStageConfig{
			StageConfig: StageConfig{
				RunningMode:      StageRunningModePool,
				MaxWorkers:       runtime.NumCPU(),
				TargetQueueDepth: 64,
			},
		},
		ROB: &ROBStageConfig{
			ResetTimeout: 100 * time.Millisecond,
		},
		CAN: &CANStageConfig{
			StageConfig: StageConfig{
				RunningMode:      StageRunningModePool,
				MaxWorkers:       runtime.NumCPU(),
				TargetQueueDepth: 64,
			},
			DBCFilePath: "/app/can/bus.dbc",
		},
		CANMessageHandler: &CANMessageHandlerStageConfig{
			StageConfig: StageConfig{
				RunningMode:      StageRunningModePool,
				MaxWorkers:       runtime.NumCPU(),
				TargetQueueDepth: 64,
			},
		},
		QuestDB: &QuestDBStageConfig{
			StageConfig: StageConfig{
				RunningMode:      StageRunningModePool,
				MaxWorkers:       runtime.NumCPU(),
				TargetQueueDepth: 64,
			},
			Address: "localhost:9000",
		},
		Telemetry: &TelemetryConfig{
			CollectorEndpoint: "localhost:4317",
		},
	}
}

type Config struct {
	ServiceName       string                        `yaml:"service_name" env:"SERVICE_NAME"`
	ConnectorSize     uint32                        `yaml:"connector_size" env:"CONNECTOR_SIZE"`
	UDP               *UDPStageConfig               `yaml:"udp" envPrefix:"UDP_"`
	Cannelloni        *CannelloniStageConfig        `yaml:"cannelloni" envPrefix:"CANNELLONI_"`
	ROB               *ROBStageConfig               `yaml:"rob" envPrefix:"ROB_"`
	CAN               *CANStageConfig               `yaml:"can" envPrefix:"CAN_"`
	CANMessageHandler *CANMessageHandlerStageConfig `yaml:"can_message_handler" envPrefix:"CAN_MESSAGE_HANDLER_"`
	QuestDB           *QuestDBStageConfig           `yaml:"quest_db" envPrefix:"QUEST_DB_"`
	Telemetry         *TelemetryConfig              `yaml:"telemetry" envPrefix:"TELEMETRY_"`
}

type StageRunningMode string

const (
	StageRunningModeSingle StageRunningMode = "single"
	StageRunningModePool   StageRunningMode = "pool"
)

type UDPStageConfig struct {
	IPAddr string `yaml:"ip_addr" env:"IP_ADDR"`
	Port   uint16 `yaml:"port" env:"PORT"`
}

func (c *UDPStageConfig) GetStageConfig() *ingress.UDPConfig {
	stageCfg := ingress.DefaultUDPConfig()

	stageCfg.IPAddr = c.IPAddr
	stageCfg.Port = c.Port

	return stageCfg
}

type StageConfig struct {
	RunningMode      StageRunningMode `yaml:"running_mode" env:"RUNNING_MODE"`
	MaxWorkers       int              `yaml:"max_workers" env:"MAX_WORKERS"`
	TargetQueueDepth int              `yaml:"target_queue_depth" env:"TARGET_QUEUE_DEPTH"`
}

type CannelloniStageConfig struct {
	StageConfig
}

func (c *CannelloniStageConfig) GetStageConfig() *processor.CannelloniConfig {
	if c.RunningMode == StageRunningModeSingle {
		return processor.DefaultCannelloniConfig(goccia.StageRunningModeSingle)
	}

	cfg := processor.DefaultCannelloniConfig(goccia.StageRunningModePool)

	cfg.Stage.Pool.MaxWorkers = c.MaxWorkers
	cfg.Stage.Pool.QueueDepthPerWorker = c.TargetQueueDepth

	return cfg
}

type ROBStageConfig struct {
	ResetTimeout time.Duration `yaml:"reset_timeout" env:"RESET_TIMEOUT"`
}

func (c *ROBStageConfig) GetStageConfig() *processor.ROBConfig {
	stageCfg := processor.DefaultROBConfig()

	stageCfg.ResetTimeout = c.ResetTimeout

	return stageCfg
}

type CANStageConfig struct {
	StageConfig

	DBCFilePath string `yaml:"dbc_file_path" env:"DBC_FILE_PATH"`
}

func (c *CANStageConfig) GetStageConfig() *processor.CANConfig {
	if c.RunningMode == StageRunningModeSingle {
		return processor.DefaultCANConfig(goccia.StageRunningModeSingle)
	}

	stageCfg := processor.DefaultCANConfig(goccia.StageRunningModePool)

	stageCfg.Stage.Pool.MaxWorkers = c.MaxWorkers
	stageCfg.Stage.Pool.QueueDepthPerWorker = c.TargetQueueDepth

	stageCfg.Messages = c.getMessages()

	return stageCfg
}

func (c *CANStageConfig) getMessages() []*acmelib.Message {
	messages := []*acmelib.Message{}

	dbcFile, err := os.Open(c.DBCFilePath)
	if err != nil {
		log.Print("failed to open dbc file: ", err)
		return messages
	}
	defer dbcFile.Close()

	bus, err := acmelib.ImportDBCFile("bus", dbcFile)
	if err != nil {
		log.Print("failed to import dbc file: ", err)
		return messages
	}

	for _, nodeInt := range bus.NodeInterfaces() {
		for _, msg := range nodeInt.SentMessages() {
			messages = append(messages, msg)
		}
	}

	return messages
}

type CANMessageHandlerStageConfig struct {
	StageConfig
}

func (c *CANMessageHandlerStageConfig) GetStageConfig() *processor.CustomConfig {
	var stageCfg *processor.CustomConfig

	switch c.RunningMode {
	case StageRunningModeSingle:
		stageCfg = processor.DefaultCustomConfig(goccia.StageRunningModeSingle)

	case StageRunningModePool:
		stageCfg = processor.DefaultCustomConfig(goccia.StageRunningModePool)

		stageCfg.Stage.Pool.MaxWorkers = c.MaxWorkers
		stageCfg.Stage.Pool.QueueDepthPerWorker = c.TargetQueueDepth
	}

	stageCfg.Name = "can_message_handler"

	return stageCfg
}

type QuestDBStageConfig struct {
	StageConfig

	Address string `yaml:"address" env:"ADDRESS"`
}

func (c *QuestDBStageConfig) GetStageConfig() *egress.QuestDBConfig {
	var stageCfg *egress.QuestDBConfig
	switch c.RunningMode {
	case StageRunningModeSingle:
		stageCfg = egress.DefaultQuestDBConfig(goccia.StageRunningModeSingle)

	case StageRunningModePool:
		stageCfg = egress.DefaultQuestDBConfig(goccia.StageRunningModePool)

		stageCfg.Stage.Pool.MaxWorkers = c.MaxWorkers
		stageCfg.Stage.Pool.QueueDepthPerWorker = c.TargetQueueDepth
	}

	stageCfg.Address = c.Address

	return stageCfg
}

type TelemetryConfig struct {
	CollectorEndpoint string `yaml:"collector_endpoint" env:"COLLECTOR_ENDPOINT"`
}

func LoadConfig() (*Config, error) {
	config := defaultConfig()

	var data []byte
	var err error

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Print("config file does not exist at ", configPath)
		goto loadEnv
	}

	// Read config file
	data, err = os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config file
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

loadEnv:
	// Load environment variables
	if err := env.Parse(config); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	return config, nil
}
