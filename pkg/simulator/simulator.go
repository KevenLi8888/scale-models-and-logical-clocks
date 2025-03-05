package simulator

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"scale-models-and-logical-clocks/pkg/cfg"
	"scale-models-and-logical-clocks/pkg/machine"
)

// Simulator coordinates the simulation of multiple virtual machines
type Simulator struct {
	Machines []*machine.Machine
	Config   cfg.Config
	Logger   *log.Logger
}

// NewSimulator creates a new simulator
func NewSimulator(config cfg.Config) (*Simulator, error) {
	// Set up logger
	logger := log.New(os.Stdout, "[Simulator] ", log.LstdFlags)

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	return &Simulator{
		Machines: make([]*machine.Machine, 0, config.Machines),
		Config:   config,
		Logger:   logger,
	}, nil
}

// Initialize sets up the virtual machines
func (s *Simulator) Initialize() error {
	s.Logger.Println("Initializing simulator...")

	// Create machines
	for i := 0; i < s.Config.Machines; i++ {
		// Create machine with clock variation and event range max from config
		machine, err := machine.NewMachine(i, s.Config.BasePort, s.Config.LogDir,
			s.Config.ClockVariation, s.Config.EventRangeMax)
		if err != nil {
			return fmt.Errorf("failed to create machine %d: %w", i, err)
		}
		s.Machines = append(s.Machines, machine)
	}

	// Connect machines to each other
	for i, machine := range s.Machines {
		for j := 0; j < s.Config.Machines; j++ {
			if i != j {
				err := machine.ConnectToMachine(j, s.Config.BasePort)
				if err != nil {
					return fmt.Errorf("failed to connect machine %d to machine %d: %w", i, j, err)
				}
			}
		}
	}

	s.Logger.Printf("Initialized %d machines", len(s.Machines))
	return nil
}

// Start begins the simulation
func (s *Simulator) Start() {
	s.Logger.Println("Starting simulation...")

	// Start all machines
	for _, machine := range s.Machines {
		machine.Start()
	}

	// Run for specified duration
	s.Logger.Printf("Simulation running for %d seconds...", s.Config.DurationSecs)
	time.Sleep(time.Duration(s.Config.DurationSecs) * time.Second)

	// Stop all machines
	for _, machine := range s.Machines {
		go machine.Stop()
	}

	s.Logger.Println("Simulation complete")
}
