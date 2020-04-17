// SPDX-FileCopyrightText: 2018-2020 Michael Picht <mipi@fsfe.org>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package workerpool

import (
	"sync"
)

// Pool represents a worker pool. The pool mainly consists of channels that let
// it communicate with the outside (submitting tasks, receiving results etc.).
type Pool struct {
	In   chan Task     // input channel
	Out  chan Result   // output channel
	stop chan struct{} // stop channel
	done chan struct{} // done channel
}

// Task represents a task that shall be executed by the workers
type Task struct {
	Name string                        // name of task
	F    func(interface{}) interface{} // function that implements the task
	In   interface{}                   // input data of task
}

// Result represents the result of a task
type Result struct {
	Name string      // name of task
	Out  interface{} // output data of task
}

// NewPool creates a new worker pool with numWorkers number of go routines
func NewPool(numWorkers int) *Pool {
	var (
		pl Pool
		wg sync.WaitGroup
	)

	pl.In = make(chan Task)
	pl.Out = make(chan Result)
	pl.stop = make(chan struct{})
	pl.done = make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		// start worker Go routine
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-pl.stop: // receive from stop channel
					// drain input channel
					go func() {
						for range pl.In {
						}
					}()
					return

				case task, ok := <-pl.In: // receive from input channel
					if !ok {
						return
					}
					// execute task and send result to output channel
					pl.Out <- Result{
						Name: task.Name,
						Out:  task.F(task.In)}
				}
			}
		}()
	}

	// wait for all worker Go routines to be done, then clean up and report
	// "done" for entire pool
	go func() {
		wg.Wait()
		close(pl.Out)
		close(pl.done)
	}()

	return &pl
}

// Stop stops processing
func (pl *Pool) Stop() {
	close(pl.stop)
}

// Wait until pool finished
func (pl *Pool) Wait() {
	<-pl.done
}
