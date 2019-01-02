// Copyright (C) 2018 Michael Picht
//
// This file is part of workerpool.
//
// workerpool is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// workerpool is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with workerpool. If not, see <http://www.gnu.org/licenses/>.

// Package workerpool implements worker Go routines that can execute tasks
// concurrently. This approach is suited for use cases where (a) the
// number of tasks is high and thus it is impossible to execute each
// task in a separate Go routine and (b) it's possible to implement
// the task execution in a single function.
package workerpool

import (
	"sync"
)

// Pool
type Pool struct {
	In   chan Task     // input channel for tasks
	Out  chan Result   // output channel for tasks
	stop chan struct{} // stop channel
	done chan struct{} // done channel
}

type Task struct {
	Name string
	F    func(interface{}) interface{}
	In   interface{}
}

type Result struct {
	Name string
	Out  interface{}
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
