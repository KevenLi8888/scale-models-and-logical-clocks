package simulator

import (
	"fmt"
	"os"
	"testing"
	"time"

	"scale-models-and-logical-clocks/pkg/cfg"
)

func TestNewSimulator(t *testing.T) {
	config := cfg.Config{
		Machines:       3,
		BasePort:       8000,
		LogDir:         "test_logs",
		ClockVariation: 5,
		EventRangeMax:  10,
		DurationSecs:   1,
	}

	// create temporary log directory
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		t.Fatalf("Unable to create log directory: %v", err)
	}
	defer os.RemoveAll(config.LogDir)

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create simulator: %v", err)
	}

	if sim == nil {
		t.Fatal("Simulator should not be nil")
	}

	if len(sim.Machines) != 0 {
		t.Errorf("Initial machine count should be 0, but got: %d", len(sim.Machines))
	}

	if sim.Config.Machines != config.Machines {
		t.Errorf("Configuration mismatch, expected: %v, got: %v", config, sim.Config)
	}
}

func TestInitialize(t *testing.T) {
	config := cfg.Config{
		Machines:       3,
		BasePort:       8050,
		LogDir:         "test_logs",
		ClockVariation: 5,
		EventRangeMax:  10,
		DurationSecs:   1,
	}

	// create temporary log directory
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		t.Fatalf("Unable to create log directory: %v", err)
	}
	defer os.RemoveAll(config.LogDir)

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create simulator: %v", err)
	}

	err = sim.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize simulator: %v", err)
	}

	if len(sim.Machines) != config.Machines {
		t.Errorf("Machine count mismatch, expected: %d, got: %d", config.Machines, len(sim.Machines))
	}

	// check if each machine is initialized correctly
	for i, machine := range sim.Machines {
		if machine.ID != i {
			t.Errorf("Machine ID mismatch, expected: %d, got: %d", i, machine.ID)
		}
	}
}

func TestCustomizeSimulation(t *testing.T) {
	config := cfg.Config{
		Machines:       3,
		BasePort:       8080,
		LogDir:         "test_logs",
		ClockVariation: 5,
		EventRangeMax:  10,
		DurationSecs:   1,
	}

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create simulator: %v", err)
	}

	newClockVariation := 10
	newEventRangeMax := 20

	sim.CustomizeSimulation(newClockVariation, newEventRangeMax)

	if sim.Config.ClockVariation != newClockVariation {
		t.Errorf("Clock variation mismatch, expected: %d, got: %d", newClockVariation, sim.Config.ClockVariation)
	}

	if sim.Config.EventRangeMax != newEventRangeMax {
		t.Errorf("Event range max mismatch, expected: %d, got: %d", newEventRangeMax, sim.Config.EventRangeMax)
	}
}

func TestStartSimulation(t *testing.T) {
	// this test uses a shorter duration
	config := cfg.Config{
		Machines:       2,
		BasePort:       8100,
		LogDir:         "test_logs",
		ClockVariation: 5,
		EventRangeMax:  10,
		DurationSecs:   1, // run for 1 second
	}

	// create temporary log directory
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		t.Fatalf("Unable to create log directory: %v", err)
	}
	defer os.RemoveAll(config.LogDir)

	sim, err := NewSimulator(config)
	if err != nil {
		t.Fatalf("Failed to create simulator: %v", err)
	}

	err = sim.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize simulator: %v", err)
	}

	// use a channel to ensure the simulation is complete
	done := make(chan struct{})
	go func() {
		sim.Start()
		close(done)
	}()

	// wait for the simulation to complete or timeout
	select {
	case <-done:
		// simulation completed normally
	case <-time.After(time.Duration(config.DurationSecs+2) * time.Second):
		t.Fatal("Simulation timed out")
	}

	// check if the log files are created
	for i := 0; i < config.Machines; i++ {
		logPath := fmt.Sprintf("%s/machine_%d.log", config.LogDir, i)
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("Machine %d log file not created: %s", i, logPath)
		}
	}
}
