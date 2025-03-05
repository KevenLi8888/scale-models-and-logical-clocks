package machine

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewMachine(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test machine creation
	m, err := NewMachine(1, 8000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Verify machine properties
	if m.ID != 1 {
		t.Errorf("Expected ID 1, got %d", m.ID)
	}
	if m.Port != 8001 {
		t.Errorf("Expected port 8001, got %d", m.Port)
	}
	if m.ClockRate < 1 || m.ClockRate > 5 {
		t.Errorf("Clock rate %d outside expected range [1-5]", m.ClockRate)
	}
	if m.EventRangeMax != 10 {
		t.Errorf("Expected EventRangeMax 10, got %d", m.EventRangeMax)
	}

	// Verify log file was created
	logPath := filepath.Join(tempDir, "machine_1.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created at %s", logPath)
	}
}

func TestMachineConnection(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	basePort := 9000

	// Create two machines
	m1, err := NewMachine(1, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 1: %v", err)
	}
	defer m1.Stop()

	m2, err := NewMachine(2, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 2: %v", err)
	}
	defer m2.Stop()

	// Start both machines
	m1.Start()
	m2.Start()

	// Connect m1 to m2
	err = m1.ConnectToMachine(2, basePort)
	if err != nil {
		t.Fatalf("Failed to connect machine 1 to machine 2: %v", err)
	}

	// Verify connection was established
	m1.connMutex.Lock()
	_, ok := m1.Connections[2]
	m1.connMutex.Unlock()
	if !ok {
		t.Errorf("Connection from machine 1 to machine 2 not found in connections map")
	}
}

func TestMessageSending(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	basePort := 10000

	// Create two machines
	m1, err := NewMachine(1, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 1: %v", err)
	}
	defer m1.Stop()

	m2, err := NewMachine(2, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 2: %v", err)
	}
	defer m2.Stop()

	// Start both machines
	m1.Start()
	m2.Start()

	// Connect m1 to m2
	err = m1.ConnectToMachine(2, basePort)
	if err != nil {
		t.Fatalf("Failed to connect machine 1 to machine 2: %v", err)
	}

	// Manually send a message from m1 to m2
	m1.sendMessage(2, false)

	// Give some time for the message to be processed
	time.Sleep(100 * time.Millisecond)

	// Check if m2 received the message
	m2.queueMu.Lock()
	queueLen := len(m2.MessageQueue)
	m2.queueMu.Unlock()

	// We can't guarantee the message is still in the queue (it might have been processed),
	// but we can check the logical clock was updated
	m2.clockMu.Lock()
	clockValue := m2.LogicalClock
	m2.clockMu.Unlock()

	// Either the message is in the queue or it was processed and updated the clock
	if queueLen == 0 && clockValue == 0 {
		t.Errorf("Message was not received by machine 2")
	}
}

func TestMulticastMessage(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	basePort := 11000

	// Create three machines
	m1, err := NewMachine(1, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 1: %v", err)
	}
	defer m1.Stop()

	m2, err := NewMachine(2, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 2: %v", err)
	}
	defer m2.Stop()

	m3, err := NewMachine(3, basePort, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine 3: %v", err)
	}
	defer m3.Stop()

	// Start all machines
	m1.Start()
	m2.Start()
	m3.Start()

	// Connect m1 to m2 and m3
	err = m1.ConnectToMachine(2, basePort)
	if err != nil {
		t.Fatalf("Failed to connect machine 1 to machine 2: %v", err)
	}
	err = m1.ConnectToMachine(3, basePort)
	if err != nil {
		t.Fatalf("Failed to connect machine 1 to machine 3: %v", err)
	}

	// Give some time for connections to stabilize
	time.Sleep(500 * time.Millisecond)

	// Send multicast from m1 to all connected machines
	m1.sendToAllMachines()

	// Give more time for the messages to be processed
	// This needs to be longer because:
	// 1. Messages need to be sent over the network
	// 2. Receiving machines need to process them in their tick cycle
	time.Sleep(1 * time.Second)

	// Check if both m2 and m3 received messages
	m2.clockMu.Lock()
	m2Clock := m2.LogicalClock
	m2.clockMu.Unlock()

	m3.clockMu.Lock()
	m3Clock := m3.LogicalClock
	m3.clockMu.Unlock()

	if m2Clock == 0 && m3Clock == 0 {
		t.Errorf("Multicast message was not received by any machine")
	}
}

func TestProcessTick(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a machine with a fixed event range for deterministic testing
	// Setting EventRangeMax to 1 ensures internal events (case > 3)
	m, err := NewMachine(1, 12000, tempDir, 5, 1)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Test internal event (case > 3, but we've set EventRangeMax to 1)
	initialClock := m.LogicalClock

	// Force an internal event by setting a seed that will generate a value > 3
	// This ensures the test is deterministic
	rand.Seed(42) // Use a fixed seed that produces a value > 3

	m.processTick()

	if m.LogicalClock <= initialClock {
		t.Errorf("Logical clock was not incremented after processTick()")
	}

	// Test message processing
	testMsg := Message{
		SenderID:    2,
		LogicalTime: m.LogicalClock + 5, // Higher than current clock
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Add message to queue
	m.queueMu.Lock()
	m.MessageQueue = append(m.MessageQueue, testMsg)
	m.queueMu.Unlock()

	// Process the tick which should handle the message
	clockBeforeMsg := m.LogicalClock
	m.processTick()

	// Check if logical clock was updated according to Lamport's algorithm
	if m.LogicalClock <= clockBeforeMsg {
		t.Errorf("Logical clock was not properly updated after receiving message")
	}
	if m.LogicalClock != testMsg.LogicalTime+1 {
		t.Errorf("Expected logical clock to be %d, got %d", testMsg.LogicalTime+1, m.LogicalClock)
	}

	// Check if message was removed from queue
	m.queueMu.Lock()
	queueLen := len(m.MessageQueue)
	m.queueMu.Unlock()
	if queueLen != 0 {
		t.Errorf("Message was not removed from queue after processing")
	}
}

func TestWriteJSON(t *testing.T) {
	// Create a server and client connection pair for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Create a machine
	m := &Machine{
		ID: 1,
	}

	// Create a test message
	testMsg := Message{
		SenderID:    1,
		LogicalTime: 42,
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Start a goroutine to read the message from the server side
	errCh := make(chan error, 1)
	msgCh := make(chan Message, 1)
	go func() {
		var receivedMsg Message
		decoder := json.NewDecoder(server)
		err := decoder.Decode(&receivedMsg)
		if err != nil {
			errCh <- fmt.Errorf("failed to decode message: %w", err)
			return
		}
		msgCh <- receivedMsg
	}()

	// Write the message to the client side
	err := m.writeJSON(client, testMsg)
	if err != nil {
		t.Fatalf("Failed to write JSON: %v", err)
	}

	// Wait for the message to be received or an error
	select {
	case err := <-errCh:
		t.Fatalf("Error reading message: %v", err)
	case receivedMsg := <-msgCh:
		// Verify the message was received correctly
		if receivedMsg.SenderID != testMsg.SenderID {
			t.Errorf("Expected SenderID %d, got %d", testMsg.SenderID, receivedMsg.SenderID)
		}
		if receivedMsg.LogicalTime != testMsg.LogicalTime {
			t.Errorf("Expected LogicalTime %d, got %d", testMsg.LogicalTime, receivedMsg.LogicalTime)
		}
		if receivedMsg.Timestamp != testMsg.Timestamp {
			t.Errorf("Expected Timestamp %s, got %s", testMsg.Timestamp, receivedMsg.Timestamp)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

func TestReadJSON(t *testing.T) {
	// Create a server and client connection pair for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// Create a test message
	testMsg := Message{
		SenderID:    2,
		LogicalTime: 42,
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Start a goroutine to write the message to the server side
	go func() {
		encoder := json.NewEncoder(server)
		err := encoder.Encode(testMsg)
		if err != nil {
			t.Errorf("Failed to encode message: %v", err)
		}
	}()

	// Read directly from the client connection instead of creating an unused machine
	var receivedMsg Message
	decoder := json.NewDecoder(client)
	err := decoder.Decode(&receivedMsg)
	if err != nil {
		t.Fatalf("Failed to decode message: %v", err)
	}

	// Verify the message was received correctly
	if receivedMsg.SenderID != testMsg.SenderID {
		t.Errorf("Expected SenderID %d, got %d", testMsg.SenderID, receivedMsg.SenderID)
	}
	if receivedMsg.LogicalTime != testMsg.LogicalTime {
		t.Errorf("Expected LogicalTime %d, got %d", testMsg.LogicalTime, receivedMsg.LogicalTime)
	}
	if receivedMsg.Timestamp != testMsg.Timestamp {
		t.Errorf("Expected Timestamp %s, got %s", testMsg.Timestamp, receivedMsg.Timestamp)
	}
}

func TestMachineGetRandomTargetID(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a machine
	m, err := NewMachine(1, 13000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Test with no connections
	targetID := m.getRandomTargetID()
	if targetID != -1 {
		t.Errorf("Expected -1 for no connections, got %d", targetID)
	}

	// Add some mock connections
	m.connMutex.Lock()
	m.Connections[2] = &mockConn{}
	m.Connections[3] = &mockConn{}
	m.connMutex.Unlock()

	// Test with connections
	targetID = m.getRandomTargetID()
	if targetID != 2 && targetID != 3 {
		t.Errorf("Expected target ID to be 2 or 3, got %d", targetID)
	}
}

func TestMachineHandleConnection(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a machine
	m, err := NewMachine(1, 14000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	// Start the machine to initialize the WaitGroup properly
	m.Start()

	// Create a server and client connection pair for testing
	server, client := net.Pipe()
	defer client.Close()

	// Manually add to the WaitGroup before calling handleConnection
	m.wg.Add(1)

	// Start a goroutine to handle the connection
	go m.handleConnection(server)

	// Create a test message
	testMsg := Message{
		SenderID:    2,
		LogicalTime: 42,
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Send the message
	encoder := json.NewEncoder(client)
	err = encoder.Encode(testMsg)
	if err != nil {
		t.Fatalf("Failed to encode message: %v", err)
	}

	// Give some time for the message to be processed
	time.Sleep(100 * time.Millisecond)

	// Check if the message was added to the queue
	m.queueMu.Lock()
	queueLen := len(m.MessageQueue)
	var receivedMsg Message
	if queueLen > 0 {
		receivedMsg = m.MessageQueue[0]
	}
	m.queueMu.Unlock()

	// Close the client connection to terminate the handleConnection goroutine
	client.Close()

	// Wait a bit for the goroutine to finish
	time.Sleep(100 * time.Millisecond)

	// Now it's safe to stop the machine
	m.Stop()

	if queueLen == 0 {
		t.Errorf("Message was not added to the queue")
	} else {
		// Verify the message was received correctly
		if receivedMsg.SenderID != testMsg.SenderID {
			t.Errorf("Expected SenderID %d, got %d", testMsg.SenderID, receivedMsg.SenderID)
		}
		if receivedMsg.LogicalTime != testMsg.LogicalTime {
			t.Errorf("Expected LogicalTime %d, got %d", testMsg.LogicalTime, receivedMsg.LogicalTime)
		}
	}
}

// Mock connection for testing
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, io.EOF }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestMachineInternalEvent(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a machine with a fixed event range for deterministic testing
	// Setting EventRangeMax to 1 ensures only internal events (case > 3)
	m, err := NewMachine(1, 15000, tempDir, 5, 1)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Test internal event
	initialClock := m.LogicalClock
	m.processTick()

	if m.LogicalClock != initialClock+1 {
		t.Errorf("Expected logical clock to be incremented by 1, got %d (initial: %d)",
			m.LogicalClock, initialClock)
	}
}

func TestMachineWithInvalidEventRange(t *testing.T) {
	// Create a temporary directory for logs
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a machine with an invalid event range (less than 3)
	m, err := NewMachine(1, 16000, tempDir, 5, 2)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Verify the event range was corrected to the default
	if m.EventRangeMax != 10 {
		t.Errorf("Expected EventRangeMax to be corrected to 10, got %d", m.EventRangeMax)
	}
}

func TestMachineCreationWithErrors(t *testing.T) {
	// Test with invalid log directory
	_, err := NewMachine(1, 8000, "/nonexistent/directory/that/should/fail", 5, 10)
	if err == nil {
		t.Errorf("Expected error when creating machine with invalid log directory, got nil")
	}

	// Create a temporary file instead of directory to cause an error
	tempFile, err := os.CreateTemp("", "not_a_directory")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = NewMachine(1, 8000, tempFile.Name(), 5, 10)
	if err == nil {
		t.Errorf("Expected error when creating machine with file as log directory, got nil")
	}

	// Test with port that's already in use
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a listener on the port we want to use
	listener, err := net.Listen("tcp", "127.0.0.1:8001")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Try to create a machine with the same port
	_, err = NewMachine(1, 8000, tempDir, 5, 10)
	if err == nil {
		t.Errorf("Expected error when creating machine with port already in use, got nil")
	}
}

func TestConnectToSelf(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewMachine(1, 17000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Try to connect to self
	err = m.ConnectToMachine(1, 17000)
	if err != nil {
		t.Errorf("Expected no error when connecting to self, got: %v", err)
	}

	// Verify no connection was added
	m.connMutex.Lock()
	if len(m.Connections) != 0 {
		t.Errorf("Expected no connections after connecting to self, got %d", len(m.Connections))
	}
	m.connMutex.Unlock()
}

func TestConnectToNonExistentMachine(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewMachine(1, 18000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Try to connect to a machine that doesn't exist
	err = m.ConnectToMachine(999, 18000)
	if err == nil {
		t.Errorf("Expected error when connecting to non-existent machine, got nil")
	}
}

func TestSendMessageNoConnection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewMachine(1, 19000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Try to send a message to a machine that we're not connected to
	initialClock := m.LogicalClock
	m.sendMessage(999, false)

	// Clock should still increment even if send fails
	if m.LogicalClock != initialClock+1 {
		t.Errorf("Expected logical clock to increment after failed send, got %d (initial: %d)",
			m.LogicalClock, initialClock)
	}
}

func TestWriteJSONError(t *testing.T) {
	// Create a server and client connection pair for testing
	server, client := net.Pipe()

	// Close the connection immediately to cause a write error
	server.Close()
	client.Close()

	// Create a machine
	m := &Machine{
		ID: 1,
	}

	// Create a test message
	testMsg := Message{
		SenderID:    1,
		LogicalTime: 42,
		Timestamp:   time.Now().Format(time.StampMilli),
	}

	// Try to write to the closed connection
	err := m.writeJSON(client, testMsg)
	if err == nil {
		t.Errorf("Expected error when writing to closed connection, got nil")
	}
}

func TestHandleConnectionErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewMachine(1, 20000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	// Start the machine to initialize the WaitGroup properly
	m.Start()
	defer m.Stop()

	// Create a server and client connection pair for testing
	server, client := net.Pipe()

	// Manually add to the WaitGroup before calling handleConnection
	m.wg.Add(1)

	// Start a goroutine to handle the connection
	go m.handleConnection(server)

	// Send invalid JSON to trigger a decode error
	_, err = client.Write([]byte("this is not valid JSON"))
	if err != nil {
		t.Fatalf("Failed to write to connection: %v", err)
	}

	// Give some time for the error to be processed
	time.Sleep(100 * time.Millisecond)

	// Close the client connection
	client.Close()

	// Wait a bit for the goroutine to finish
	time.Sleep(100 * time.Millisecond)
}

func TestSendToAllMachinesWithError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewMachine(1, 21000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}
	defer m.Stop()

	// Add a closed connection to simulate an error when sending
	server, client := net.Pipe()
	client.Close()
	server.Close()

	m.connMutex.Lock()
	m.Connections[2] = server
	m.connMutex.Unlock()

	// Initial clock value
	initialClock := m.LogicalClock

	// Send to all machines (should handle the error gracefully)
	m.sendToAllMachines()

	// Clock should still increment
	if m.LogicalClock != initialClock+1 {
		t.Errorf("Expected logical clock to increment after multicast with error, got %d (initial: %d)",
			m.LogicalClock, initialClock)
	}
}

func TestListenForConnectionsStop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "machine_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m, err := NewMachine(1, 22000, tempDir, 5, 10)
	if err != nil {
		t.Fatalf("Failed to create machine: %v", err)
	}

	// Start the machine
	m.Start()

	// Stop it after a short time
	time.Sleep(100 * time.Millisecond)
	m.Stop()

	// This test mainly checks that listenForConnections exits cleanly when m.running is set to false
}
