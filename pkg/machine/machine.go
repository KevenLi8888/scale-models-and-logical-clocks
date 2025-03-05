package machine

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Message represents a message sent between machines
type Message struct {
	SenderID    int    `json:"sender_id"`
	LogicalTime int    `json:"logical_time"`
	Timestamp   string `json:"timestamp"`
}

// Machine represents a virtual machine in the distributed system
type Machine struct {
	ID            int
	ClockRate     int              // Number of ticks per second
	LogicalClock  int              // Current logical clock value
	MessageQueue  []Message        // Queue for incoming messages
	Connections   map[int]net.Conn // Connections to other machines (map from machine ID to connection)
	Listener      net.Listener     // Listener for incoming TCP connections
	Port          int              // Port this machine listens on
	Logger        *log.Logger      // Logger for this machine
	LogFile       *os.File         // Log file
	queueMu       sync.Mutex       // Mutex for protecting MessageQueue
	clockMu       sync.Mutex       // Mutex for protecting LogicalClock
	running       bool             // Flag to indicate if machine is running
	wg            sync.WaitGroup   // WaitGroup to wait for all goroutines to finish
	EventRangeMax int              // Maximum value for random event generation (1-N)
	connMutex     sync.Mutex       // Mutex for protecting connections map
}

// NewMachine creates a new virtual machine
func NewMachine(id int, basePort int, logDir string, clockVariation int, eventRangeMax int) (*Machine, error) {
	// Create log directory if it doesn't exist
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	logFile, err := os.Create(filepath.Join(logDir, fmt.Sprintf("machine_%d.log", id)))
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger := log.New(logFile, fmt.Sprintf("[Machine %d] ", id), log.LstdFlags|log.Lmicroseconds)

	// Generate random clock rate between 1 and clockVariation
	clockRate := rand.Intn(clockVariation) + 1
	port := basePort + id

	// Ensure eventRangeMax is at least 3
	if eventRangeMax < 3 {
		eventRangeMax = 10 // Default to 10 if less than 3
	}

	// Create TCP listener
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		logFile.Close()
		return nil, fmt.Errorf("failed to create TCP listener: %w", err)
	}

	machine := &Machine{
		ID:            id,
		ClockRate:     clockRate,
		LogicalClock:  0,
		MessageQueue:  make([]Message, 0),
		Connections:   make(map[int]net.Conn),
		Listener:      listener,
		Port:          port,
		Logger:        logger,
		LogFile:       logFile,
		running:       false,
		EventRangeMax: eventRangeMax,
	}

	logger.Printf("Created machine with clock rate: %d ticks per second, event range: 1-%d",
		clockRate, eventRangeMax)
	return machine, nil
}

// ConnectToMachine connects this machine to another machine
func (m *Machine) ConnectToMachine(targetID int, basePort int) error {
	if targetID == m.ID {
		return nil // Don't connect to self
	}

	targetPort := basePort + targetID
	// Give the other machine time to start listening
	var conn net.Conn
	var err error

	// Try to connect multiple times with exponential backoff
	maxRetries := 5
	for retry := 0; retry < maxRetries; retry++ {
		conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", targetPort))
		if err == nil {
			break
		}
		backoffDuration := time.Duration(100*(1<<retry)) * time.Millisecond
		time.Sleep(backoffDuration)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to machine %d: %w", targetID, err)
	}

	m.connMutex.Lock()
	m.Connections[targetID] = conn
	m.connMutex.Unlock()

	m.Logger.Printf("Connected to machine %d at port %d", targetID, targetPort)
	return nil
}

// Start starts the machine's operation
func (m *Machine) Start() {
	m.running = true

	// Start listener goroutine
	m.wg.Add(1)
	go m.listenForConnections()

	// Start main execution loop
	m.wg.Add(1)
	go m.runMainLoop()

	m.Logger.Printf("Machine %d started", m.ID)
}

// Stop gracefully stops the machine
func (m *Machine) Stop() {
	fmt.Println("Stopping machine", m.ID)
	m.running = false
	m.wg.Wait()

	// Close connections
	m.connMutex.Lock()
	for _, conn := range m.Connections {
		conn.Close()
	}
	m.connMutex.Unlock()

	m.Listener.Close()
	m.LogFile.Close()

	m.Logger.Printf("Machine %d stopped", m.ID)
}

// listenForConnections accepts incoming TCP connections
func (m *Machine) listenForConnections() {
	defer m.wg.Done()

	// Set a deadline for accepting connections so we can check if we should stop
	for m.running {
		m.Listener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))

		conn, err := m.Listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if m.running { // Only log error if still running
				m.Logger.Printf("Error accepting TCP connection: %v", err)
			}
			continue
		}

		// Handle the connection in a new goroutine
		m.wg.Add(1)
		go m.handleConnection(conn)
	}
}

// handleConnection handles an incoming TCP connection
func (m *Machine) handleConnection(conn net.Conn) {
	defer conn.Close()
	defer m.wg.Done()

	decoder := json.NewDecoder(conn)

	for m.running {
		// Read and decode message
		var msg Message
		err := decoder.Decode(&msg)

		if err != nil {
			if err == io.EOF || !m.running {
				// Connection closed or machine stopping
				return
			}
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, try again
				continue
			}

			m.Logger.Printf("Error decoding message: %v", err)
			return
		}

		// fmt.Println("Received message:", msg)

		// Add message to queue - now using queueMu instead of mu
		m.queueMu.Lock()
		m.MessageQueue = append(m.MessageQueue, msg)
		m.queueMu.Unlock()
	}
}

// runMainLoop runs the main execution loop of the machine
func (m *Machine) runMainLoop() {
	defer m.wg.Done()

	tickDuration := time.Second / time.Duration(m.ClockRate)
	ticker := time.NewTicker(tickDuration)
	defer ticker.Stop()

	for m.running {
		<-ticker.C
		m.processTick()
	}
}

// processTick processes a single clock tick
func (m *Machine) processTick() {
	// We need both locks since we're potentially operating on both resources
	m.queueMu.Lock()

	// If there's a message in the queue, process it
	if len(m.MessageQueue) > 0 {
		msg := m.MessageQueue[0]
		m.MessageQueue = m.MessageQueue[1:]

		// Release queue lock as early as possible
		m.queueMu.Unlock()

		// Now acquire clock lock for updating logical clock
		m.clockMu.Lock()

		// Update logical clock according to Lamport's algorithm
		if msg.LogicalTime > m.LogicalClock {
			m.LogicalClock = msg.LogicalTime
		}
		m.LogicalClock++

		// Get the current clock value before releasing lock
		currentClock := m.LogicalClock
		m.clockMu.Unlock()

		// Get current queue length - need to lock again briefly
		m.queueMu.Lock()
		queueLen := len(m.MessageQueue)
		m.queueMu.Unlock()

		// Log the message receipt
		m.Logger.Printf("RECEIVE from Machine %d | System time: %s | Queue length: %d | Logical clock: %d",
			msg.SenderID, time.Now().Format(time.StampMilli), queueLen, currentClock)
	} else {
		// No message in queue, release queue lock
		m.queueMu.Unlock()

		// No message, generate random number in range of 1 to EventRangeMax
		randVal := rand.Intn(m.EventRangeMax) + 1

		// Determine action based on random value per requirements
		switch randVal {
		case 1:
			// Send to a machine with odd ID
			if len(m.Connections) > 0 {
				targetID := m.getRandomOddTargetID()
				if targetID != -1 {
					m.sendMessage(targetID, false)
				}
			}
		case 2:
			// Send to a machine with even ID
			if len(m.Connections) > 0 {
				targetID := m.getRandomEvenTargetID()
				if targetID != -1 {
					m.sendMessage(targetID, false)
				}
			}
		case 3:
			// Send to all other machines
			m.sendToAllMachines()
		default:
			// Internal event (any value > 3)
			m.clockMu.Lock()
			m.LogicalClock++
			currentClock := m.LogicalClock
			m.clockMu.Unlock()

			m.Logger.Printf("INTERNAL EVENT | System time: %s | Logical clock: %d",
				time.Now().Format(time.StampMilli), currentClock)
		}
	}
}

// getRandomTargetID gets a random machine ID from the connections map
func (m *Machine) getRandomTargetID() int {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()

	if len(m.Connections) == 0 {
		return -1
	}

	// Get all connected machine IDs
	ids := make([]int, 0, len(m.Connections))
	for id := range m.Connections {
		ids = append(ids, id)
	}

	// Select a random ID
	return ids[rand.Intn(len(ids))]
}

// getRandomOddTargetID gets a random machine ID with odd value from the connections map
func (m *Machine) getRandomOddTargetID() int {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()

	// Get all connected machine IDs that are odd
	oddIDs := make([]int, 0)
	for id := range m.Connections {
		if id%2 != 0 { // Odd ID
			oddIDs = append(oddIDs, id)
		}
	}

	if len(oddIDs) == 0 {
		return -1
	}

	// Select a random odd ID
	return oddIDs[rand.Intn(len(oddIDs))]
}

// getRandomEvenTargetID gets a random machine ID with even value from the connections map
func (m *Machine) getRandomEvenTargetID() int {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()

	// Get all connected machine IDs that are even
	evenIDs := make([]int, 0)
	for id := range m.Connections {
		if id%2 == 0 { // Even ID
			evenIDs = append(evenIDs, id)
		}
	}

	if len(evenIDs) == 0 {
		return -1
	}

	// Select a random even ID
	return evenIDs[rand.Intn(len(evenIDs))]
}

// sendToAllMachines sends a message to all connected machines
func (m *Machine) sendToAllMachines() {
	m.connMutex.Lock()
	defer m.connMutex.Unlock()

	// Update logical clock once for the multicast event
	m.clockMu.Lock()
	m.LogicalClock++
	currentClock := m.LogicalClock
	m.clockMu.Unlock()

	// Prepare message
	msg := Message{
		SenderID:    m.ID,
		LogicalTime: currentClock,
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Send to all machines
	for id, conn := range m.Connections {
		err := m.writeJSON(conn, msg)
		if err != nil {
			m.Logger.Printf("Error sending multicast to machine %d: %v", id, err)
		}
	}

	// Log the multicast
	m.Logger.Printf("MULTICAST | System time: %s | Logical clock: %d",
		time.Now().Format(time.StampMilli), currentClock)
}

// sendMessage sends a message to another machine
func (m *Machine) sendMessage(targetID int, multicast bool) {
	// Update logical clock
	m.clockMu.Lock()
	m.LogicalClock++
	currentClock := m.LogicalClock
	m.clockMu.Unlock()

	// Get connection
	m.connMutex.Lock()
	conn, ok := m.Connections[targetID]
	m.connMutex.Unlock()

	if !ok {
		m.Logger.Printf("Error: No connection to machine %d", targetID)
		return
	}

	// Prepare message
	msg := Message{
		SenderID:    m.ID,
		LogicalTime: currentClock,
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Send the message
	err := m.writeJSON(conn, msg)
	if err != nil {
		m.Logger.Printf("Error sending message to machine %d: %v", targetID, err)
		return
	}

	// Log the send
	messageType := "SEND"
	if multicast {
		messageType = "MULTICAST"
	}
	m.Logger.Printf("%s to Machine %d | System time: %s | Logical clock: %d",
		messageType, targetID, time.Now().Format(time.StampMilli), currentClock)
}

// writeJSON writes a JSON message to a connection
func (m *Machine) writeJSON(conn net.Conn, msg Message) error {
	encoder := json.NewEncoder(conn)
	return encoder.Encode(msg)
}
