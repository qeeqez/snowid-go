# Snowflake

A Go library that implements Discord-style Snowflake IDs with a custom epoch starting from January 1st, 2024. It generates unique 64-bit IDs that are time-sortable and guaranteed to be unique within a distributed system.

## Structure

The ID is composed of:
- 42 bits for timestamp (milliseconds since 2024-01-01 00:00:00 UTC)
- 5 bits for worker ID (0-31)
- 5 bits for process ID (0-31)
- 12 bits for sequence number

## Usage

```go
package main

import (
    "fmt"
    "github.com/qeeqez/snowflake"
)

func main() {
    // Create a new snowflake generator with worker ID 1 and process ID 1
    sf, err := snowflake.NewSnowflake(1, 1)
    if err != nil {
        panic(err)
    }

    // Generate a new ID
    id, err := sf.NextID()
    if err != nil {
        panic(err)
    }
    fmt.Printf("Generated ID: %d\n", id)

    // Parse an ID to get its components
    timestamp, workerID, processID, sequence := snowflake.Parse(id)
    fmt.Printf("Timestamp: %d\n", timestamp)
    fmt.Printf("Worker ID: %d\n", workerID)
    fmt.Printf("Process ID: %d\n", processID)
    fmt.Printf("Sequence: %d\n", sequence)
}
```

## Features

- Thread-safe ID generation
- Custom epoch (2024-01-01)
- Discord-style bit allocation
- ID parsing functionality
- Guaranteed unique IDs within a distributed system (when properly configured)

## Installation

```bash
go get github.com/qeeqez/snowflake
```

## License

MIT License
