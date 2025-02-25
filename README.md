# ❄️ SnowID Go

[![Go Reference](https://pkg.go.dev/badge/github.com/qeeqez/snowid.svg)](https://pkg.go.dev/github.com/qeeqez/snowid)
[![Go Report Card](https://goreportcard.com/badge/github.com/qeeqez/snowid)](https://goreportcard.com/report/github.com/qeeqez/snowid)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> A Go implementation of a Snowflake-like ID generator with 42-bit timestamp.

**Generate 64-bit unique identifiers that are:**

- ⚡️ Fast (~200ns per ID)
- 📈 Time-sorted
- 🔄 Monotonic
- 🔒 Thread-safe
- 🌐 Distributed-ready
- 🎯 Zero allocations

## 🧮 ID Structure

**Example ID**: 151819733950271234

**Default configuration:**

```text
|------------------------------------------|------------|------------|
|           TIMESTAMP (42 bits)            | NODE (10)  |  SEQ (12)  |
|------------------------------------------|------------|------------|
```

- Timestamp: 42 bits = 139 years from 2024-01-01 (1704067200000)
- Node ID: 10 bits = 1,024 nodes
- Sequence: 12 bits = 4,096 IDs/ms/node

## 📊 Performance & Comparisons

### Social Media Platform Configurations

| Platform  | Timestamp | Node Bits | Sequence Bits | Max Nodes |
|-----------|-----------|-----------|---------------|-----------|
| Sonyflake | 38        | 16        | 8             | 65,535    |
| Twitter   | 41        | 10        | 12            | 1,024     |
| Instagram | 41        | 13        | 10            | 8,192     |
| Discord   | 42        | 10        | 12            | 1,024     |
| SnowID Go | 42        | 10        | 12            | 1,024     |

### Node vs Sequence Bits Trade-off

| Node Bits | Max Nodes | IDs/ms/node | Time/ID |
|-----------|-----------|-------------|---------|
| 6         | 64        | 65,536      | ~20ns   |
| 8         | 256       | 16,384      | ~60ns   |
| 10        | 1,024     | 4,096       | ~200ns  |
| 12        | 4,096     | 1,024       | ~800ns  |
| 14        | 16,384    | 256         | ~3.2µs  |
| 16        | 65,536    | 64          | ~12.8µs |

Choose configuration based on your needs:

- More nodes → Increase node bits (max 16 bits = 65,536 nodes)
- More IDs per node → Increase sequence bits (min 6 node bits = 64 nodes)
- Total bits (node + sequence) is fixed at 22 bits

## 🎯 Quick Start

```bash
go get github.com/qeeqez/snowid
```

```go
package main

import (
	"fmt"
	"log"
	"github.com/qeeqez/snowid"
)

func main() {
	// Create a new Node with machine ID 1
	node, err := snowid.NewNode(1)
	if err != nil {
		log.Fatal(err)
	}

	// Generate a new ID
	id, err := node.Generate()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Generated ID: %d\n", id)
}
```

## 🔧 Configuration

```go
package main

import (
	"fmt"
	"log"
	"time"
	"github.com/qeeqez/snowid"
)

func main() {
	// Create a Node with custom epoch
	customEpoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	node, err := snowid.NewNodeWithEpoch(1, customEpoch)
	if err != nil {
		log.Fatal(err)
	}

	// Generate and decompose ID
	id, _ := node.Generate()
	parts := node.Decompose(id)

	fmt.Printf("ID: %d\n", id)
	fmt.Printf("Timestamp: %d\n", parts.Timestamp)
	fmt.Printf("Machine ID: %d\n", parts.MachineID)
	fmt.Printf("Sequence: %d\n", parts.Sequence)

	// Get generation time
	t := node.Time(id)
	fmt.Printf("Generated at: %s\n", t.UTC().Format(time.RFC3339Nano))
}
```

### ℹ️ Available Methods

```go
// Create a new Node
node, err := snowid.NewNode(1)

// Generate a new ID
id, err := node.Generate()

// Extract components
parts := node.Decompose(id) // Get all components at once
timestamp := parts.Timestamp // Get timestamp from ID
machineID := parts.MachineID    // Get machine ID from ID
sequence := parts.Sequence      // Get sequence from ID

// Get generation time
time := node.Time(id) // Convert ID to time.Time

// Configuration information
maxMachineID := node.MaxMachineID() // Get maximum allowed machine ID (1023)
```

## 🔍 Error Handling

The library provides specific error types for different scenarios:

- `ErrTimeMovedBackwards`: When system time moves backwards
- `ErrMachineIDTooLarge`: When machine ID is > 1023
- `ErrSequenceOverflow`: When sequence number is exhausted
- `ErrInvalidEpoch`: When epoch is set to a future time

## 🚀 Examples

Check out the [examples](./example) directory for:

- Basic usage
- Custom configuration
- Concurrent generation
- Performance benchmarks

### Thread-Safe Concurrent Usage

```go
package main

import (
	"fmt"
	"log"
	"sync"
	"github.com/qeeqez/snowid"
)

func main() {
	// Create a shared node
	node, _ := snowid.NewNode(1)

	// Generate IDs concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := node.Generate()
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}
			fmt.Printf("Generated ID: %d\n", id)
		}()
	}
	wg.Wait()
}
```

## 📜 License

MIT - See [LICENSE](LICENSE) for details

## 🙏 Acknowledgments

- Twitter's [Snowflake](https://github.com/twitter-archive/snowflake)
- Discord's [Snowflake](https://discord.com/developers/docs/reference#snowflakes)
- [SnowID Rust](https://github.com/qeeqez/snowid)
