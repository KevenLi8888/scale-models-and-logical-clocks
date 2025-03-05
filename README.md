# Scale Models and Logical Clocks

This project implements a distributed system model with logical clocks. Each virtual machine runs at a randomly assigned speed (clock rate) and communicates with other virtual machines in the system.

## Project Overview

This simulation models multiple machines running at different speeds, with each machine:

1. Running at a random clock rate
2. Maintaining a logical clock (Lamport's clock)
3. Having a message queue for incoming messages
4. Communicating with other machines through TCP connections using JSON messages

Each virtual machine can either process a message from its queue, send messages to other machines, or perform internal operations based on probabilistic logic.

## Communication Protocol

The system uses TCP for reliable communication between machines. Messages are exchanged in JSON format with the following structure:

```json
{
  "sender_id": 1,
  "logical_time": 42,
  "timestamp": "Apr 14 15:04:05.999"
}
```

This approach provides:
- Reliable message delivery (TCP)
- Human-readable message format (JSON)
- Extensible message structure for future enhancements

## System Requirements

- Go 1.22.1 or higher

## Configuration

The system can be configured through the `cfg/cfg.yaml` file:

```yaml
machines: 3         # Number of virtual machines to simulate
basePort: 8000      # Base port for communication
logDir: "logs"      # Directory to store log files
durationSecs: 60    # Duration to run the simulation in seconds
internalEventProb: 7  # Probability of internal event (7 of 10)
clockVariation: 6     # Maximum clock tick variation (1-6)
```

## Running the Simulation

### Basic Run

To run a simulation with the default configuration:

```
go run cmd/main.go
```

### Run with Command Line Options

You can override configuration settings using command-line arguments:

```
go run cmd/main.go -duration=120 -clockvar=3 -eventmax=5
```

## Analyzing Results

Logs for each virtual machine are stored in the `logs/[timestamp]/` directory. Each log contains entries showing:
- Type of event (INTERNAL EVENT, SEND, RECEIVE)
- System time
- Message queue length (for RECEIVE events)
- Logical clock value

Analyze these logs to observe:
- Jumps in logical clock values
- Clock drift between machines
- Impact of different timing configurations
- Message queue growth patterns

## Lab Notebook

When running experiments, observe and note:
1. How the logical clock values jump in each machine
2. The drift between logical clocks on different machines
3. How the message queue length varies over time
4. How different clock speeds and internal event probabilities impact system behavior