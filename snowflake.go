package snowflake

import (
	"errors"
	"sync"
	"time"
)

const (
	// Custom epoch (2024-01-01 00:00:00 UTC)
	epoch int64 = 1704067200000

	// Bits allocations (following Discord's format)
	timestampBits  uint8 = 42
	workerIDBits   uint8 = 5
	processIDBits  uint8 = 5
	sequenceBits   uint8 = 12

	// Maximum values
	maxWorkerID   int64 = -1 ^ (-1 << workerIDBits)
	maxProcessID  int64 = -1 ^ (-1 << processIDBits)
	maxSequence   int64 = -1 ^ (-1 << sequenceBits)

	// Bit shifts
	timestampShift = workerIDBits + processIDBits + sequenceBits
	workerShift    = processIDBits + sequenceBits
	processShift   = sequenceBits
)

// Snowflake holds the necessary information to generate snowflake IDs
type Snowflake struct {
	mu        sync.Mutex
	timestamp int64
	workerID  int64
	processID int64
	sequence  int64
}

// NewSnowflake creates a new Snowflake instance
func NewSnowflake(workerID, processID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, errors.New("worker ID must be between 0 and 31")
	}
	if processID < 0 || processID > maxProcessID {
		return nil, errors.New("process ID must be between 0 and 31")
	}

	return &Snowflake{
		workerID:  workerID,
		processID: processID,
	}, nil
}

// NextID generates a new snowflake ID
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().UnixMilli()

	if timestamp < epoch {
		return 0, errors.New("invalid system clock: time is before epoch")
	}

	timestamp = timestamp - epoch

	if s.timestamp == timestamp {
		s.sequence = (s.sequence + 1) & maxSequence
		if s.sequence == 0 {
			// Sequence overflow, wait for next millisecond
			for timestamp <= s.timestamp {
				timestamp = time.Now().UnixMilli() - epoch
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = timestamp

	id := (timestamp << timestampShift) |
		(s.workerID << workerShift) |
		(s.processID << processShift) |
		s.sequence

	return id, nil
}

// Parse deconstructs a snowflake ID into its components
func Parse(id int64) (timestamp, workerID, processID, sequence int64) {
	timestamp = (id >> timestampShift) + epoch
	workerID = (id >> workerShift) & maxWorkerID
	processID = (id >> processShift) & maxProcessID
	sequence = id & maxSequence
	return
}
