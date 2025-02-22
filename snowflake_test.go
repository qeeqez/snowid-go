package snowflake

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	tests := []struct {
		name      string
		machineID int64
		wantErr   bool
	}{
		{"valid machine ID", 123, false},
		{"zero machine ID", 0, false},
		{"max machine ID", maxMachineID, false},
		{"negative machine ID", -1, true},
		{"too large machine ID", maxMachineID + 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNode(tt.machineID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if node.machineID != tt.machineID {
					t.Errorf("machine ID = %v, want %v", node.machineID, tt.machineID)
				}
			}
		})
	}
}

func TestNewNodeWithEpoch(t *testing.T) {
	futureTime := time.Now().Add(time.Hour)
	pastTime := time.Now().Add(-time.Hour)

	tests := []struct {
		name      string
		machineID int64
		epoch     time.Time
		wantErr   bool
	}{
		{"valid epoch", 123, pastTime, false},
		{"future epoch", 123, futureTime, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := NewNodeWithEpoch(tt.machineID, tt.epoch)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !node.epoch.Equal(tt.epoch) {
					t.Errorf("epoch = %v, want %v", node.epoch, tt.epoch)
				}
			}
		})
	}
}

func TestNode_Generate(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	// Generate multiple IDs
	var ids []int64
	for i := 0; i < 100; i++ {
		id, err := node.Generate()
		if err != nil {
			t.Errorf("failed to generate ID: %v", err)
		}
		ids = append(ids, id)
	}

	// Check uniqueness
	seen := make(map[int64]bool)
	for _, id := range ids {
		if seen[id] {
			t.Error("generated duplicate ID")
		}
		seen[id] = true
	}

	// Check monotonicity
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Error("IDs are not monotonically increasing")
		}
	}
}

func TestNode_GenerateConcurrent(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	var wg sync.WaitGroup
	idChan := make(chan int64, 1000)
	workers := 10
	idsPerWorker := 100

	// Generate IDs concurrently
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerWorker; j++ {
				id, err := node.Generate()
				if err != nil {
					t.Errorf("failed to generate ID: %v", err)
					return
				}
				idChan <- id
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()
	close(idChan)

	// Check uniqueness
	seen := make(map[int64]bool)
	var ids []int64
	for id := range idChan {
		if seen[id] {
			t.Error("generated duplicate ID")
		}
		seen[id] = true
		ids = append(ids, id)
	}

	// Verify we got the expected number of IDs
	expectedCount := workers * idsPerWorker
	if len(ids) != expectedCount {
		t.Errorf("got %d IDs, want %d", len(ids), expectedCount)
	}
}

func TestNode_GenerateHighConcurrency(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	var wg sync.WaitGroup
	idChan := make(chan int64, 100000)
	workers := 100
	idsPerWorker := 1000

	start := time.Now()
	// Generate IDs concurrently
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerWorker; j++ {
				id, err := node.Generate()
				if err != nil {
					t.Errorf("failed to generate ID: %v", err)
					return
				}
				idChan <- id
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()
	close(idChan)

	// Check uniqueness
	seen := make(map[int64]bool)
	var ids []int64
	for id := range idChan {
		if seen[id] {
			t.Errorf("duplicate ID found: %d", id)
		}
		seen[id] = true
		ids = append(ids, id)
	}

	duration := time.Since(start)
	t.Logf("Generated %d unique IDs in %v (%.2f IDs/sec)", len(ids), duration, float64(len(ids))/duration.Seconds())

	// Verify we got the expected number of IDs
	expectedCount := workers * idsPerWorker
	if len(ids) != expectedCount {
		t.Errorf("got %d IDs, want %d", len(ids), expectedCount)
	}
}

func TestNode_Decompose(t *testing.T) {
	node, err := NewNode(123)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	id, err := node.Generate()
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}

	decomposed := node.Decompose(id)

	// Check machine ID
	if decomposed.MachineID != 123 {
		t.Errorf("machine ID = %v, want 123", decomposed.MachineID)
	}

	// Check sequence (should be 0 or small number)
	if decomposed.Sequence < 0 || decomposed.Sequence > maxSequence {
		t.Errorf("sequence = %v, should be between 0 and %v", decomposed.Sequence, maxSequence)
	}

	// Check timestamp
	idTime := node.Time(id)
	now := time.Now()
	if idTime.After(now) || now.Sub(idTime) > time.Second {
		t.Errorf("timestamp = %v, should be close to now (%v)", idTime, now)
	}
}

func BenchmarkNode_Generate(b *testing.B) {
	node, err := NewNode(1)
	if err != nil {
		b.Fatalf("failed to create node: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := node.Generate()
		if err != nil {
			b.Fatalf("failed to generate ID: %v", err)
		}
	}
}

func BenchmarkNode_GenerateParallel(b *testing.B) {
	node, err := NewNode(1)
	if err != nil {
		b.Fatalf("failed to create node: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := node.Generate()
			if err != nil {
				b.Fatalf("failed to generate ID: %v", err)
			}
		}
	})
}

func TestNode_TimestampBoundaries(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	// Test maximum timestamp (42 bits)
	maxTimestamp := int64((uint64(1) << 42) - 1) // Use uint64 for correct bit operation
	id := node.createID(maxTimestamp, 0)
	decomposed := node.Decompose(id)
	if decomposed.Timestamp != maxTimestamp {
		t.Errorf("max timestamp = %v, want %v", decomposed.Timestamp, maxTimestamp)
	}

	// Test zero timestamp
	id = node.createID(0, 0)
	decomposed = node.Decompose(id)
	if decomposed.Timestamp != 0 {
		t.Errorf("zero timestamp = %v, want 0", decomposed.Timestamp)
	}
}

func TestNode_SequenceOverflow(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	// Set sequence to max value
	atomic.StoreInt64(&node.sequence, maxSequence)
	
	// Generate should still work by waiting for next millisecond
	id, err := node.Generate()
	if err != nil {
		t.Errorf("failed to generate ID after sequence overflow: %v", err)
	}

	// Decompose and verify sequence was reset
	decomposed := node.Decompose(id)
	if decomposed.Sequence > maxSequence {
		t.Errorf("sequence = %v, want <= %v", decomposed.Sequence, maxSequence)
	}
}

func TestNode_TimeAccuracy(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	now := time.Now().UTC()
	id, err := node.Generate()
	if err != nil {
		t.Fatalf("failed to generate ID: %v", err)
	}

	idTime := node.Time(id)
	timeDiff := idTime.Sub(now)

	// Should be within reasonable bounds (1 second)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("time difference too large: %v", timeDiff)
	}
}

func TestNode_IDComponents(t *testing.T) {
	tests := []struct {
		timestamp int64
		machineID int64
		sequence  int64
	}{
		{0, 0, 0},                    // All zeros
		{int64((uint64(1) << 42) - 1), 0, 0},           // Max timestamp
		{0, maxMachineID, 0},        // Max machine ID
		{0, 0, maxSequence},         // Max sequence
		{int64((uint64(1) << 42) - 1), maxMachineID, maxSequence}, // All max values
		{1234567, 789, 4000},        // Random values
	}

	node, err := NewNode(0)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	for _, tt := range tests {
		id := node.createID(tt.timestamp, tt.sequence)
		decomposed := node.Decompose(id)

		if decomposed.Timestamp != tt.timestamp {
			t.Errorf("timestamp = %v, want %v", decomposed.Timestamp, tt.timestamp)
		}
		if decomposed.Sequence != tt.sequence {
			t.Errorf("sequence = %v, want %v", decomposed.Sequence, tt.sequence)
		}
	}
}

func TestNode_CustomEpochEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		epoch     time.Time
		wantErr   bool
	}{
		{
			"epoch at Unix zero time",
			time.Unix(0, 0),
			false,
		},
		{
			"epoch one millisecond before now",
			time.Now().Add(-time.Millisecond),
			false,
		},
		{
			"epoch exactly now",
			time.Now(),
			false,
		},
		{
			"epoch one millisecond in future",
			time.Now().Add(time.Millisecond),
			true,
		},
		{
			"epoch far in future",
			time.Now().AddDate(1, 0, 0),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNodeWithEpoch(1, tt.epoch)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNodeWithEpoch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNode_ClockDrift(t *testing.T) {
	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	epochMs := node.epoch.UnixNano() / millisecond

	// Test 1: Large clock drift should return error
	t.Run("large drift", func(t *testing.T) {
		now := time.Now().UTC().UnixNano() / millisecond
		timestamp := now - epochMs
		atomic.StoreInt64(&node.time, timestamp+10) // Set 10ms drift

		id, err := node.Generate()
		if err != ErrTimeMovedBackwards {
			t.Errorf("expected ErrTimeMovedBackwards, got %v (id=%v)", err, id)
		}
		if id != 0 {
			t.Errorf("expected id = 0 on error, got %v", id)
		}
	})

	// Test 2: Small clock drift (1ms) should work
	t.Run("small drift", func(t *testing.T) {
		now := time.Now().UTC().UnixNano() / millisecond
		timestamp := now - epochMs
		atomic.StoreInt64(&node.time, timestamp+1) // Set 1ms drift
		t.Logf("Current timestamp: %v, Stored timestamp: %v", timestamp, timestamp+1)

		id, err := node.Generate()
		if err != nil {
			t.Errorf("failed to generate ID with small drift: %v (id=%v)", err, id)
			return
		}

		// Verify ID components
		decomposed := node.Decompose(id)
		if decomposed.MachineID != 1 {
			t.Errorf("machine ID = %v, want 1 (id=%v)", decomposed.MachineID, id)
		}
		if decomposed.Timestamp != timestamp+1 {
			t.Errorf("timestamp = %v, want %v (id=%v)", decomposed.Timestamp, timestamp+1, id)
		}
	})

	// Test 3: Normal operation after resetting time
	t.Run("normal operation", func(t *testing.T) {
		now := time.Now().UTC().UnixNano() / millisecond
		timestamp := now - epochMs
		atomic.StoreInt64(&node.time, timestamp)
		t.Logf("Set timestamp to: %v", timestamp)

		id, err := node.Generate()
		if err != nil {
			t.Errorf("failed to generate ID in normal operation: %v (id=%v)", err, id)
			return
		}

		decomposed := node.Decompose(id)
		if decomposed.MachineID != 1 {
			t.Errorf("machine ID = %v, want 1 (id=%v)", decomposed.MachineID, id)
		}
		if decomposed.Timestamp < timestamp {
			t.Errorf("timestamp = %v, should be >= %v (id=%v)", decomposed.Timestamp, timestamp, id)
		}
	})
}

func TestNode_GenerateStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	node, err := NewNode(1)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	workers := 50
	idsPerWorker := 10000
	idChan := make(chan int64, workers*idsPerWorker)
	errChan := make(chan error, workers)

	var wg sync.WaitGroup
	start := time.Now()

	// Generate IDs concurrently
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerWorker; j++ {
				id, err := node.Generate()
				if err != nil {
					errChan <- err
					return
				}
				idChan <- id
			}
		}()
	}

	// Wait for completion or error
	go func() {
		wg.Wait()
		close(idChan)
		close(errChan)
	}()

	// Check for errors
	for err := range errChan {
		t.Errorf("error during stress test: %v", err)
	}

	// Collect and verify IDs
	seen := make(map[int64]bool)
	var ids []int64
	for id := range idChan {
		if seen[id] {
			t.Error("generated duplicate ID")
		}
		seen[id] = true
		ids = append(ids, id)
	}

	duration := time.Since(start)
	idsPerSec := float64(len(ids)) / duration.Seconds()
	t.Logf("Generated %d unique IDs in %v (%.2f IDs/sec)", len(ids), duration, idsPerSec)

	// Verify expected count
	expectedCount := workers * idsPerWorker
	if len(ids) != expectedCount {
		t.Errorf("got %d IDs, want %d", len(ids), expectedCount)
	}
}
