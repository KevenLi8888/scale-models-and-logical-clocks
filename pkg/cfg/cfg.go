package cfg

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Machines     int    `yaml:"machines"`     // Number of virtual machines
	BasePort     int    `yaml:"basePort"`     // Base port for communication
	LogDir       string `yaml:"logDir"`       // Directory to store log files
	DurationSecs int    `yaml:"durationSecs"` // Duration to run the simulation in seconds
	// Parameters for experiment variations
	EventRangeMax  int `yaml:"eventRangeMax"`  // Maximum value for random event generation (must be >= 3)
	ClockVariation int `yaml:"clockVariation"` // Clock variation (1-6)
}

func ParseConfig(configPath string) (Config, error) {
	// Read the config file
	file, err := os.Open(configPath)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	// Parse the config file
	var cfg Config
	err = yaml.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return Config{}, err
	}

	// Default values if not specified
	if cfg.Machines == 0 {
		fmt.Println("No machines specified, defaulting to 3")
		cfg.Machines = 3
	}
	if cfg.BasePort == 0 {

		fmt.Println("No base port specified, defaulting to 8000")
		cfg.BasePort = 8000
	}
	if cfg.LogDir == "" {
		fmt.Println("No log directory specified, defaulting to 'logs'")
		cfg.LogDir = "logs"
	}
	if cfg.DurationSecs == 0 {
		fmt.Println("No duration specified, defaulting to 60 seconds")
		cfg.DurationSecs = 60 // 1 minute default
	}
	if cfg.EventRangeMax == 0 || cfg.EventRangeMax < 3 {
		fmt.Println("Invalid event range max specified, defaulting to 10")
		cfg.EventRangeMax = 10 // Default to 1-10 range
	}
	if cfg.ClockVariation == 0 {
		fmt.Println("Invalid clock variation specified, defaulting to 6")
		cfg.ClockVariation = 6 // 1-6 range for clock speeds
	}

	return cfg, nil
}
