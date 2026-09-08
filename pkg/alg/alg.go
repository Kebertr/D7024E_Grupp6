package alg

// NumberDoubler takes numbers from an input channel,
// doubles each number, and sends the result to an output channel
func NumberDoubler(input <-chan int, output chan<- int) {
    // Process each number that comes in
    for number := range input {
        doubled := number * 2
        output <- doubled
    }
}