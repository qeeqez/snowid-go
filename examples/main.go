package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/qeeqez/snowid"
)

func main() {
	// Create a new Node with machine ID 1
	node, err := snowid.NewNode(1)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Basic Usage ===")
	// Generate a new ID
	id, err := node.Generate()
	if err != nil {
		log.Fatal(err)
	}

	// Print the ID
	fmt.Printf("Generated ID: %d\n", id)

	// Decompose the ID into its components
	parts := node.Decompose(id)
	fmt.Printf("Components:\n")
	fmt.Printf("  Timestamp: %d\n", parts.Timestamp)
	fmt.Printf("  Machine ID: %d\n", parts.MachineID)
	fmt.Printf("  Sequence: %d\n", parts.Sequence)

	// Get the time at which the ID was generated
	t := node.Time(id)
	fmt.Printf("Generated at: %s\n", t.Format(time.RFC3339Nano))

	fmt.Println("\n=== Custom Epoch ===")
	// Create a Node with custom epoch
	customEpoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	customNode, err := snowid.NewNodeWithEpoch(1, customEpoch)
	if err != nil {
		log.Fatal(err)
	}

	// Generate an ID with custom epoch
	customID, err := customNode.Generate()
	if err != nil {
		log.Fatal(err)
	}

	customParts := customNode.Decompose(customID)
	customTime := customNode.Time(customID)
	fmt.Printf("ID with custom epoch: %d\n", customID)
	fmt.Printf("Generated at: %s\n", customTime.Format(time.RFC3339Nano))
	fmt.Printf("Time since epoch: %v\n", customTime.Sub(customEpoch))
	fmt.Printf("Components:\n")
	fmt.Printf("  Timestamp: %d\n", customParts.Timestamp)
	fmt.Printf("  Machine ID: %d\n", customParts.MachineID)
	fmt.Printf("  Sequence: %d\n", customParts.Sequence)

	fmt.Println("\n=== Concurrent Generation ===")
	// Demonstrate concurrent ID generation
	var wg sync.WaitGroup
	idChan := make(chan int64, 10)

	// Generate 10 IDs concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := node.Generate()
			if err != nil {
				log.Printf("Error generating ID: %v", err)
				return
			}
			idChan <- id
		}()
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(idChan)
	}()

	// Collect and print IDs
	var ids []int64
	for id := range idChan {
		ids = append(ids, id)
	}

	// Sort and print IDs
	fmt.Printf("Generated %d unique IDs:\n", len(ids))
	for i, id := range ids {
		parts := node.Decompose(id)
		t := node.Time(id)
		fmt.Printf("%2d: ID: %d, Time: %s, Seq: %d\n",
			i+1, id, t.Format("15:04:05.000"), parts.Sequence)
	}

	fmt.Println("\n=== Performance Test ===")
	// Simple performance test
	count := 100000
	start := time.Now()
	for i := 0; i < count; i++ {
		_, err := node.Generate()
		if err != nil {
			log.Fatal(err)
		}
	}
	duration := time.Since(start)
	opsPerSec := float64(count) / duration.Seconds()

	fmt.Printf("Generated %d IDs in %v (%.2f IDs/sec)\n",
		count, duration, opsPerSec)
}
