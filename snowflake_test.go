package snowflake

import (
	"sync"
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
