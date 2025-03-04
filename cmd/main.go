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
	runMultiple := flag.Bool("multiple", false, "Run multiple experiments with different parameters")
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

	if *runMultiple {
		// Run multiple experiments with different parameters
		runMultipleExperiments(config)
	} else {
		// Run a single experiment
		runSingleExperiment(config)
	}
}

func runSingleExperiment(config cfg.Config) {
	fmt.Println("=== Starting Single Experiment ===")
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
	fmt.Println("Experiment completed. Logs saved to:", config.LogDir)
}

func runMultipleExperiments(baseConfig cfg.Config) {
	// First experiment: Default settings (high variation, high event range)
	experiment1Config := baseConfig
	experiment1Config.LogDir = filepath.Join(baseConfig.LogDir, "exp1_high_var_high_range")
	fmt.Println("\n=== Experiment 1: High clock variation (1-6), High event range (1-10) ===")

	sim1, _ := simulator.NewSimulator(experiment1Config)
	sim1.Initialize()
	sim1.Start()

	// Second experiment: Low clock variation, high event range
	experiment2Config := baseConfig
	experiment2Config.ClockVariation = 2 // Lower variation (1-2)
	experiment2Config.LogDir = filepath.Join(baseConfig.LogDir, "exp2_low_var_high_range")
	fmt.Println("\n=== Experiment 2: Low clock variation (1-2), High event range (1-10) ===")

	sim2, _ := simulator.NewSimulator(experiment2Config)
	sim2.Initialize()
	sim2.Start()

	// Third experiment: High clock variation, low event range
	experiment3Config := baseConfig
	experiment3Config.EventRangeMax = 5 // Lower event range (1-5)
	experiment3Config.LogDir = filepath.Join(baseConfig.LogDir, "exp3_high_var_low_range")
	fmt.Println("\n=== Experiment 3: High clock variation (1-6), Low event range (1-5) ===")

	sim3, _ := simulator.NewSimulator(experiment3Config)
	sim3.Initialize()
	sim3.Start()

	// Fourth experiment: Low clock variation, low event range
	experiment4Config := baseConfig
	experiment4Config.ClockVariation = 2 // Lower variation (1-2)
	experiment4Config.EventRangeMax = 5  // Lower event range (1-5)
	experiment4Config.LogDir = filepath.Join(baseConfig.LogDir, "exp4_low_var_low_range")
	fmt.Println("\n=== Experiment 4: Low clock variation (1-2), Low event range (1-5) ===")

	sim4, _ := simulator.NewSimulator(experiment4Config)
	sim4.Initialize()
	sim4.Start()

	fmt.Println("\nAll experiments completed. Results saved in:", baseConfig.LogDir)
}
