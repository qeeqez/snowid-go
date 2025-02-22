// Package snowflake implements a distributed unique ID generator inspired by Twitter's Snowflake
// but with extended 42-bit timestamp like Discord for longer epoch time.
//
// A Snowflake ID is composed of:
//   - 42 bits for time in milliseconds (gives us 139 years)
//   - 10 bits for machine id (gives us 1024 machines)
//   - 12 bits for sequence number (4096 unique IDs per millisecond per machine)
package snowflake

import (
	"errors"
	"sync/atomic"
	"time"
)

const (
	// Bit lengths of Snowflake ID parts
	timestampBits uint8 = 42 // Extended from Twitter's 41 bits to Discord's 42 bits
	machineIDBits uint8 = 10
	sequenceBits  uint8 = 12

	// Max values for Snowflake ID parts
	maxMachineID = int64(-1) ^ (int64(-1) << machineIDBits) // 1023
	maxSequence  = int64(-1) ^ (int64(-1) << sequenceBits)  // 4095

	// Bit shifts for composing Snowflake ID
	timestampLeftShift = machineIDBits + sequenceBits
	machineIDShift    = sequenceBits
)

var (
	// Errors
	ErrTimeMovedBackwards = errors.New("time has moved backwards")
	ErrMachineIDTooLarge  = errors.New("machine ID must be between 0 and 1023")
	ErrSequenceOverflow   = errors.New("sequence overflow")
	ErrInvalidEpoch       = errors.New("epoch must be a time in the past")

	// Default epoch is set to 2024-01-01 00:00:00 UTC
	defaultEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
)

// Node represents a snowflake generator node/machine
type Node struct {
	epoch     time.Time
	machineID int64
	time      int64
	sequence  int64
}

// NewNode creates a new snowflake node that can generate unique IDs
func NewNode(machineID int64) (*Node, error) {
	return NewNodeWithEpoch(machineID, defaultEpoch)
}

// NewNodeWithEpoch creates a new snowflake node with custom epoch
func NewNodeWithEpoch(machineID int64, epoch time.Time) (*Node, error) {
	if machineID < 0 || machineID > maxMachineID {
		return nil, ErrMachineIDTooLarge
	}

	if epoch.After(time.Now()) {
		return nil, ErrInvalidEpoch
	}

	return &Node{
		epoch:     epoch,
		machineID: machineID,
		time:      0,
		sequence:  0,
	}, nil
}

// Generate creates and returns a unique snowflake ID
func (n *Node) Generate() (int64, error) {
	var timestamp, seq int64

	for {
		now := time.Now().UTC()
		timestamp = now.Sub(n.epoch).Milliseconds()

		t := atomic.LoadInt64(&n.time)
		if timestamp < t {
			// Clock moved backwards, wait until we catch up
			time.Sleep(time.Duration(t-timestamp) * time.Millisecond)
			continue
		}

		if t == timestamp {
			seq = atomic.AddInt64(&n.sequence, 1) - 1 // Subtract 1 since AddInt64 returns the new value
			if seq > maxSequence {
				// Sequence exhausted, wait for next millisecond
				time.Sleep(time.Until(now.Add(time.Millisecond)))
				continue
			}
		} else {
			// Try to update timestamp and reset sequence
			if atomic.CompareAndSwapInt64(&n.time, t, timestamp) {
				seq = 0
				atomic.StoreInt64(&n.sequence, 1) // Start from 1 for next sequence
			} else {
				continue // CAS failed, retry
			}
		}

		return n.createID(timestamp, seq), nil
	}
}

// createID composes a 64-bit snowflake ID from timestamp, machineID and sequence
func (n *Node) createID(timestamp, sequence int64) int64 {
	id := timestamp << timestampLeftShift
	id |= n.machineID << machineIDShift
	id |= sequence
	return id
}

// Decompose breaks down a snowflake ID into its components
type ID struct {
	Timestamp int64
	MachineID int64
	Sequence  int64
}

// Decompose extracts the timestamp, machine ID and sequence from a snowflake ID
func (n *Node) Decompose(id int64) ID {
	return ID{
		Timestamp: (id >> timestampLeftShift),
		MachineID: (id >> machineIDShift) & maxMachineID,
		Sequence:  id & maxSequence,
	}
}

// Time returns the time at which the snowflake ID was generated
func (n *Node) Time(id int64) time.Time {
	decomposed := n.Decompose(id)
	return n.epoch.Add(time.Duration(decomposed.Timestamp) * time.Millisecond)
}
