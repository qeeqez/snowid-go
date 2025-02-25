// Package snowid implements a distributed unique ID generator inspired by Twitter's Snowflake
// but with extended 42-bit timestamp like Discord for longer epoch time.
//
// A SnowID is composed of:
//   - 42 bits for time in milliseconds (gives us 139 years)
//   - 10 bits for machine id (gives us 1024 machines)
//   - 12 bits for sequence number (4096 unique IDs per millisecond per machine)
package snowid

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

const (
	// Bit lengths of SnowID ID parts
	timestampBits uint8 = 42 // Extended from Twitter's 41 bits to Discord's 42 bits
	machineIDBits uint8 = 10
	sequenceBits  uint8 = 12

	// Max values for SnowID ID parts
	maxMachineID = int64(-1) ^ (int64(-1) << machineIDBits) // 1023
	maxSequence  = int64(-1) ^ (int64(-1) << sequenceBits)  // 4095

	// Bit shifts for composing SnowID ID
	timestampLeftShift = machineIDBits + sequenceBits
	machineIDShift     = sequenceBits

	// Time constants
	millisecond = int64(time.Millisecond / time.Nanosecond)

	// Pre-calculated masks and limits
	timestampMask = uint64((1 << timestampBits) - 1)
	machineIDMask = uint64((1 << machineIDBits) - 1)
	sequenceMask  = uint64((1 << sequenceBits) - 1)
	maxTimestamp  = 1 << timestampBits
)

var (
	ErrTimeMovedBackwards = errors.New("time has moved backwards")
	ErrMachineIDTooLarge  = errors.New("machine ID must be between 0 and 1023")
	ErrSequenceOverflow   = errors.New("sequence overflow")
	ErrInvalidEpoch       = errors.New("epoch must be a time in the past")

	// Default epoch is set to 2024-01-01 00:00:00 UTC
	defaultEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
)

// Node represents a snowid generator node/machine
type Node struct {
	epoch            time.Time
	epochMs          int64 // Cached epoch in milliseconds
	machineID        int64
	shiftedMachineID uint64 // Pre-shifted machine ID
	time             int64
	sequence         int64
	mockTime         *int64
}

// NewNode creates a new snowid node that can generate unique IDs
func NewNode(machineID int64) (*Node, error) {
	return NewNodeWithEpoch(machineID, defaultEpoch)
}

// NewNodeWithEpoch creates a new snowid node with custom epoch
func NewNodeWithEpoch(machineID int64, epoch time.Time) (*Node, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, ErrMachineIDTooLarge
	}

	if epoch.After(time.Now()) {
		return nil, ErrInvalidEpoch
	}

	return &Node{
		epoch:            epoch,
		epochMs:          epoch.UnixNano() / millisecond,
		machineID:        machineID,
		shiftedMachineID: (uint64(machineID) & machineIDMask) << machineIDShift,
		time:             0,
		sequence:         0,
		mockTime:         nil,
	}, nil
}

// setMockTime sets a mock time for testing purposes
func (n *Node) setMockTime(t *int64) {
	n.mockTime = t
}

// Generate creates and returns a unique snowid ID
func (n *Node) Generate() (int64, error) {
	for {
		var now int64
		if n.mockTime != nil {
			now = atomic.LoadInt64(n.mockTime)
		} else {
			now = time.Now().UTC().UnixNano() / millisecond
		}
		timestamp := now - n.epochMs

		if uint64(timestamp) >= maxTimestamp {
			return 0, fmt.Errorf("timestamp out of range: %d", timestamp)
		}

		currentTime := atomic.LoadInt64(&n.time)
		if timestamp < currentTime {
			diff := currentTime - timestamp
			if diff > 5 { // Increased tolerance for parallel execution
				return 0, ErrTimeMovedBackwards
			}
			timestamp = currentTime
		}

		var seq int64
		if timestamp == currentTime {
			seq = atomic.AddInt64(&n.sequence, 1) - 1
			if seq > maxSequence {
				if n.mockTime == nil {
					time.Sleep(250 * time.Microsecond)
				}
				continue
			}
		} else if timestamp > currentTime {
			if !atomic.CompareAndSwapInt64(&n.time, currentTime, timestamp) {
				continue
			}
			atomic.StoreInt64(&n.sequence, 1)
			seq = 0
		} else {
			continue
		}

		return n.createID(timestamp, seq), nil
	}
}

// createID composes a 64-bit snowid ID from timestamp and sequence
func (n *Node) createID(timestamp, sequence int64) int64 {
	return int64(
		(uint64(timestamp)&timestampMask)<<timestampLeftShift |
			n.shiftedMachineID |
			(uint64(sequence) & sequenceMask),
	)
}

// Decompose breaks down a snowid ID into its components
type ID struct {
	Timestamp int64
	MachineID int64
	Sequence  int64
}

// Decompose extracts the timestamp, machine ID and sequence from a snowid ID
func (n *Node) Decompose(id int64) ID {
	// Convert to uint64 for bit operations
	uid := uint64(id)

	// Extract components using masks
	return ID{
		Timestamp: int64((uid >> timestampLeftShift) & timestampMask),
		MachineID: int64((uid >> machineIDShift) & machineIDMask),
		Sequence:  int64(uid & sequenceMask),
	}
}

// Time returns the time at which the snowid ID was generated
func (n *Node) Time(id int64) time.Time {
	decomposed := n.Decompose(id)
	return n.epoch.Add(time.Duration(decomposed.Timestamp) * time.Millisecond)
}
