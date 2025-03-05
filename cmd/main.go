package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"scale-models-and-logical-clocks/pkg/cfg"
	"scale-models-and-logical-clocks/pkg/simulator"
)

func main() {
	// Define command-line flags
	configPath := flag.String("config", "cfg/cfg.yaml", "Path to configuration file")
	duration := flag.Int("duration", 0, "Duration to run the simulation (seconds, overrides config)")
	clockVar := flag.Int("clockvar", 0, "Clock variation (1-6, overrides config)")
	eventMax := flag.Int("eventmax", 0, "Maximum value for random event generation (must be >= 3, overrides config)")
	flag.Parse()

	// Load configuration
	config, err := cfg.ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	// Override config with command-line args if provided
	if *duration > 0 {
		config.DurationSecs = *duration
	}
	if *clockVar > 0 && *clockVar <= 6 {
		config.ClockVariation = *clockVar
	}
	if *eventMax >= 3 {
		config.EventRangeMax = *eventMax
	}

	// Create logs directory with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	config.LogDir = filepath.Join("logs", timestamp)
	err = os.MkdirAll(config.LogDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Run the simulation
	runSimulation(config)
}

func runSimulation(config cfg.Config) {
	fmt.Println("=== Starting Simulation ===")
	fmt.Printf("Number of machines: %d\n", config.Machines)
	fmt.Printf("Clock variation: %d\n", config.ClockVariation)
	fmt.Printf("Event range max: %d\n", config.EventRangeMax)
	fmt.Printf("Duration: %d seconds\n", config.DurationSecs)

	// Create simulator
	sim, err := simulator.NewSimulator(config)
	if err != nil {
		log.Fatalf("Failed to create simulator: %v", err)
	}

	// Initialize and run
	err = sim.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize simulator: %v", err)
	}

	sim.Start()
	fmt.Println("Simulation completed. Logs saved to:", config.LogDir)
}
