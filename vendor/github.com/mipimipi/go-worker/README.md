# go-worker

Implements worker Go routines that can execute tasks concurrently. This worker approach is suited for use cases where

* the number of tasks is high and thus it's impossible to execute each task in a separate Go routine

* and it's possible to implement the task execution in a single function.

This is the case, for instance, if you want to convert a big number of music files or pictures from one format to another.

## Usage

The package consists of only one function (`Setup`). It's called with

* the function that implements the task execution

* and the number of worker routines that shall be created.

`Setup` returns two channel:

* An input channel that is used to send the task parameters

* An output channel to receive the result of the task execution

## Example

Calculation of the first 100 Fibonacci numbers with 10 worker routines.

    package main

    import (
        "fmt"

        worker "github.com/mipimipi/go-worker"
    )

    // mapping of n to the n-th Fibonacci number
    type fibN struct {
        n   int
        fib int
    }

    // fibonacci calculates the n-th Fibonacci number
    func fibonacci(n int) fibN {
        if n < 2 {
            return fibN{n: n, fib: n}
        }
        return fibN{n: n, fib: fibonacci(n-1).fib + fibonacci(n-2).fib}
    }

    func main() {
        // get worklist channel and result channel for workers
        wl, res := worker.Setup(func(i interface{}) interface{} { return fibonacci(i.(int)) }, 10)

        // send parameters for workers to worklist channel. To not have to wait,
        // that's done in a Go routine
        go func() {
            for n := 0; n < 100; n++ {
                wl <- n
            }
            close(wl)
        }()

        // retrieve results from results channel
        for r := range res {
            fmt.Printf("%3d-th Fibonacci Number = %d\n", r.(fibN).n, r.(fibN).fib)
        }
    }
