# Snowflake ID Generator

A high-performance, thread-safe implementation of Discord's Snowflake ID format in Go.

## Features

- Fully compatible with Discord's Snowflake format
- Thread-safe using atomic operations
- Custom epoch support (default: 2024-01-01 00:00:00 UTC)
- High performance (~244ns/op sequential, ~307ns/op parallel)
- Zero memory allocations
- Comprehensive test coverage including concurrent generation

## Snowflake Format

The snowflake format follows Discord's specification:

```
111111111111111111111111111111111111111111 11111 11111 111111111111
64                                         22    17    12          0
```

- Timestamp: 42 bits (milliseconds since epoch)
- Worker ID: 5 bits (0-31)
- Process ID: 5 bits (0-31)
- Sequence: 12 bits (0-4095 per millisecond)

## Usage

```go
// Create a new snowflake generator with worker ID 1 and process ID 1
sf, err := NewSnowflake(1, 1)
if err != nil {
    log.Fatal(err)
}

// Generate a new ID
id, err := sf.NextID()
if err != nil {
    log.Fatal(err)
}

// Parse an ID back into its components
timestamp, workerID, processID, sequence := Parse(id)
```

## Thread Safety

The implementation uses atomic operations instead of mutexes for better performance. It's safe to use a single Snowflake instance across multiple goroutines.

## Error Handling

The implementation includes proper error handling for various edge cases:

- Invalid worker/process IDs
- Clock moving backwards
- Time before epoch
- Sequence exhaustion

## Performance

Benchmark results on Apple M3 Pro:

```
BenchmarkNextID/Sequential-12     4921972    244.2 ns/op    0 B/op    0 allocs/op
BenchmarkNextID/Parallel-12       3988285    306.9 ns/op    0 B/op    0 allocs/op
```

## License

MIT
