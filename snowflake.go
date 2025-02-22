package snowflake

import (
	"errors"
	"sync/atomic"
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

var (
	ErrInvalidWorkerID  = errors.New("worker ID must be between 0 and 31")
	ErrInvalidProcessID = errors.New("process ID must be between 0 and 31")
	ErrClockBackwards   = errors.New("invalid system clock: time moved backwards")
	ErrTimeBeforeEpoch  = errors.New("invalid system clock: time is before epoch")
	ErrSequenceExhausted = errors.New("sequence exhausted for current millisecond")
)

// Snowflake holds the necessary information to generate snowflake IDs
type Snowflake struct {
	workerID  int64
	processID int64
	time      atomic.Int64 // last timestamp
	seq       atomic.Int64 // sequence within the same millisecond
}

// NewSnowflake creates a new Snowflake instance
func NewSnowflake(workerID, processID int64) (*Snowflake, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, ErrInvalidWorkerID
	}
	if processID < 0 || processID > maxProcessID {
		return nil, ErrInvalidProcessID
	}

	return &Snowflake{
		workerID:  workerID,
		processID: processID,
	}, nil
}

// NextID generates a new snowflake ID using atomic operations
func (s *Snowflake) NextID() (int64, error) {
	for {
		timestamp := time.Now().UnixMilli()
		if timestamp < epoch {
			return 0, ErrTimeBeforeEpoch
		}
		timestamp = timestamp - epoch

		lastTimestamp := s.time.Load()
		if timestamp < lastTimestamp {
			return 0, ErrClockBackwards
		}

		var seq int64
		if timestamp == lastTimestamp {
			// Same millisecond, try to increment sequence
			currentSeq := s.seq.Load()
			seq = (currentSeq + 1) & maxSequence
			if seq == 0 {
				// Sequence exhausted, wait for next millisecond
				continue
			}
			if !s.seq.CompareAndSwap(currentSeq, seq) {
				// Another thread modified sequence, retry
				continue
			}
		} else {
			// Different millisecond, try to update timestamp and reset sequence
			if !s.time.CompareAndSwap(lastTimestamp, timestamp) {
				// Another thread updated timestamp, retry
				continue
			}
			s.seq.Store(0)
			seq = 0
		}

		// Compose ID
		id := (timestamp << timestampShift) |
			(s.workerID << workerShift) |
			(s.processID << processShift) |
			seq

		return id, nil
	}
}

// Parse deconstructs a snowflake ID into its components
func Parse(id int64) (timestamp, workerID, processID, sequence int64) {
	const mask = (1 << timestampBits) - 1
	timestamp = ((id >> timestampShift) & mask) + epoch
	workerID = (id >> workerShift) & maxWorkerID
	processID = (id >> processShift) & maxProcessID
	sequence = id & maxSequence
	return
}

// waitNextMillis waits until the next millisecond
func waitNextMillis(last int64) int64 {
	timestamp := time.Now().UnixMilli()
	for timestamp <= last {
		timestamp = time.Now().UnixMilli()
	}
	return timestamp
}
