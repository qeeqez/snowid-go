package snowflake

import (
	"sync"
	"testing"
	"time"
)

func TestNewSnowflake(t *testing.T) {
	tests := []struct {
		name      string
		workerID  int64
		processID int64
		wantErr   error
	}{
		{"valid IDs", 1, 1, nil},
		{"zero IDs", 0, 0, nil},
		{"max valid IDs", maxWorkerID, maxProcessID, nil},
		{"worker ID too high", 32, 1, ErrInvalidWorkerID},
		{"process ID too high", 1, 32, ErrInvalidProcessID},
		{"negative worker ID", -1, 1, ErrInvalidWorkerID},
		{"negative process ID", 1, -1, ErrInvalidProcessID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf, err := NewSnowflake(tt.workerID, tt.processID)
			if err != tt.wantErr {
				t.Errorf("NewSnowflake() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && sf == nil {
				t.Error("NewSnowflake() returned nil Snowflake with no error")
			}
		})
	}
}

func TestNextID(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("Failed to create Snowflake: %v", err)
	}

	t.Run("Sequential Generation", func(t *testing.T) {
		id1, err := sf.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}

		id2, err := sf.NextID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}

		if id1 >= id2 {
			t.Errorf("Second ID should be greater than first: id1=%d, id2=%d", id1, id2)
		}

		ts1, w1, p1, seq1 := Parse(id1)
		ts2, w2, p2, seq2 := Parse(id2)

		if w1 != 1 || p1 != 1 || w2 != 1 || p2 != 1 {
			t.Error("Worker or Process ID mismatch")
		}

		if ts1 > ts2 {
			t.Error("Second timestamp should not be less than first")
		}

		if ts1 == ts2 && seq2 <= seq1 {
			t.Error("Sequence should increment within same millisecond")
		}
	})

	t.Run("Concurrent Generation", func(t *testing.T) {
		const numGoroutines = 10
		const idsPerGoroutine = 1000
		var wg sync.WaitGroup
		ids := make([]int64, numGoroutines*idsPerGoroutine)
		
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(offset int) {
				defer wg.Done()
				for j := 0; j < idsPerGoroutine; j++ {
					id, err := sf.NextID()
					if err != nil {
						t.Errorf("Failed to generate ID: %v", err)
						return
					}
					ids[offset+j] = id
				}
			}(i * idsPerGoroutine)
		}
		
		wg.Wait()

		// Check for duplicates
		seen := make(map[int64]bool)
		for _, id := range ids {
			if seen[id] {
				t.Errorf("Duplicate ID found: %d", id)
			}
			seen[id] = true
		}
	})
}

func TestParse(t *testing.T) {
	sf, _ := NewSnowflake(1, 1)
	id, _ := sf.NextID()

	timestamp, workerID, processID, sequence := Parse(id)

	if workerID != 1 {
		t.Errorf("Expected worker ID 1, got %d", workerID)
	}

	if processID != 1 {
		t.Errorf("Expected process ID 1, got %d", processID)
	}

	now := time.Now().UnixMilli()
	if timestamp > now || timestamp < epoch {
		t.Errorf("Timestamp %d should be between epoch %d and current time %d", timestamp, epoch, now)
	}

	if sequence < 0 || sequence > maxSequence {
		t.Errorf("Sequence %d should be between 0 and %d", sequence, maxSequence)
	}

	// Test bit masking
	const testID int64 = 1<<22 | 1<<17 | 1<<12 | 1 // Test bits for worker, process and sequence
	ts, wid, pid, seq := Parse(testID)
	if wid != 1 {
		t.Errorf("Expected worker ID 1, got %d", wid)
	}
	if pid != 1 {
		t.Errorf("Expected process ID 1, got %d", pid)
	}
	if seq != 1 {
		t.Errorf("Expected sequence 1, got %d", seq)
	}
	if ts != epoch+1 {
		t.Errorf("Expected timestamp %d, got %d", epoch+1, ts)
	}
}

func BenchmarkNextID(b *testing.B) {
	sf, _ := NewSnowflake(1, 1)
	b.ResetTimer()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = sf.NextID()
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = sf.NextID()
			}
		})
	})
}
