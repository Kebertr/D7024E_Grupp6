package alg

import (
	"slices"
	"sort"
	"testing"
	"time"
)

func TestMultipleNumberDoublers(t *testing.T) {
	// Step 1: Create channels for communication
	inputChannel := make(chan int, 5)
	outputChannel := make(chan int, 5)

	// Step 2: Start 3 NumberDoublers working concurrently
	numWorkers := 3
	for i := 0; i < numWorkers; i++ {
		go NumberDoubler(inputChannel, outputChannel)
	}

	// Step 3: Send test numbers - workers will grab them unpredictably
	testNumbers := []int{1, 2, 3, 4, 5}
	for _, number := range testNumbers {
		inputChannel <- number
	}
	close(inputChannel) // Signal that we're done sending numbers

	// Step 4: Collect all results (order will be unpredictable)
	var actualResults []int
	expectedCount := len(testNumbers)

	for i := 0; i < expectedCount; i++ {
		select {
		case result := <-outputChannel:
			actualResults = append(actualResults, result)
		case <-time.After(1 * time.Second):
			t.Fatalf("Test timed out waiting for result %d", i+1)
		}
	}

	// Step 5: Sort both slices before comparing (since order is unpredictable)
	sort.Ints(actualResults)
	expectedResults := []int{2, 4, 6, 8, 10}
	sort.Ints(expectedResults) // Already sorted, but good practice

	if !slices.Equal(actualResults, expectedResults) {
		t.Errorf("Expected %v, got %v", expectedResults, actualResults)
	}
}

func TestMultipleNumberDoublersWithMap(t *testing.T) {
	inputChannel := make(chan int, 5)
	outputChannel := make(chan int, 5)

	// Start 3 NumberDoublers
	numWorkers := 3
	for i := 0; i < numWorkers; i++ {
		go NumberDoubler(inputChannel, outputChannel)
	}

	// Send test numbers
	testNumbers := []int{1, 2, 3, 4, 5}
	for _, number := range testNumbers {
		inputChannel <- number
	}
	close(inputChannel)

	// Create map of expected results
	expectedResults := map[int]bool{2: true, 4: true, 6: true, 8: true, 10: true}

	// Check each result and mark as received
	for i := 0; i < len(testNumbers); i++ {
		result := <-outputChannel
		if !expectedResults[result] {
			t.Errorf("Unexpected result: %d", result)
		} else {
			delete(expectedResults, result) // Mark as received
		}
	}

	// Check that we got all expected results
	if len(expectedResults) > 0 {
		remaining := make([]int, 0, len(expectedResults))
		for result := range expectedResults {
			remaining = append(remaining, result)
		}
		t.Errorf("Missing expected results: %v", remaining)
	}
}
