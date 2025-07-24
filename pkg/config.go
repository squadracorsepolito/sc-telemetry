package pkg

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/squadracorsepolito/acmetel/can"
	"github.com/squadracorsepolito/acmetel/cannelloni"
	"github.com/squadracorsepolito/acmetel/questdb"
	"github.com/squadracorsepolito/acmetel/udp"
)

const configDir = "/app/config/"
const configFile = "config.yaml"

type Config struct {
	ServiceName string
	Stages      *StagesConfig
	Connectors  *ConnectorsConfig
	DBCFilePath string
}

type StagesConfig struct {
	UDP        *udp.Config
	Cannelloni *cannelloni.Config
	CAN        *can.Config `yaml:"can"`
	QuestDB    *questdb.Config
}

type ConnectorsConfig struct {
	UDPSize        int
	CannelloniSize int
	CANSize        int `yaml:"canSize"`
}

func defaultConfig() *Config {
	return &Config{
		ServiceName: "sc-telemetry",
		Stages: &StagesConfig{
			UDP:        udp.NewDefaultConfig(),
			Cannelloni: cannelloni.NewDefaultConfig(),
			CAN:        can.NewDefaultConfig(),
			QuestDB:    questdb.NewDefaultConfig(),
		},
		Connectors: &ConnectorsConfig{
			UDPSize:        4096,
			CannelloniSize: 4096,
			CANSize:        4096,
		},
		DBCFilePath: "/app/can/db.dbc",
	}
}

func saveConfig(config *Config) error {
	// Marshal config file
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0644); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configDir+configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func LoadConfig() (*Config, error) {
	config := defaultConfig()

	// Check if config file exists
	if _, err := os.Stat(configDir + configFile); os.IsNotExist(err) {
		// Config file doesn't exist, save it
		if err := saveConfig(config); err != nil {
			return nil, err
		}

		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(configDir + configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config file
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return config, nil
}
