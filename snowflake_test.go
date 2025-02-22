package snowflake

import (
	"testing"
	"time"
)

func TestNewSnowflake(t *testing.T) {
	tests := []struct {
		name      string
		workerID  int64
		processID int64
		wantErr   bool
	}{
		{"valid IDs", 1, 1, false},
		{"worker ID too high", 32, 1, true},
		{"process ID too high", 1, 32, true},
		{"negative worker ID", -1, 1, true},
		{"negative process ID", 1, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSnowflake(tt.workerID, tt.processID)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSnowflake() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNextID(t *testing.T) {
	sf, err := NewSnowflake(1, 1)
	if err != nil {
		t.Fatalf("Failed to create Snowflake: %v", err)
	}

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
}
