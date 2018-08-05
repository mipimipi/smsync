// Copyright (C) 2018 Michael Picht
//
// This file is part of go-worker.
//
// smsync is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-worker is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-worker. If not, see <http://www.gnu.org/licenses/>.

// Package worker implements worker Go routines that can execute tasks
// concurrently. This approach is suited for use cases where (a) the
// number of tasks is high and thus it is impossible to execute each
// task in a separate Go routine and (b) it's possible to implement
// the task execution in a single function.
package worker

// Setup creates a number of worker Go routines that can execute tasks. These
// tasks are implemented by a single function, which is a parameter of Setup.
// Setup has the following input parameters: (1) The function that implements
// the execution of the task and (2) the number of worker Go routines that
// Setup creates. It returns two channels: (1) An input channel to pass
// parameters to the task implementation and (2) an output channel to send the
// results of the tasks execution
func Setup(task func(interface{}) interface{}, numWorkers int) (chan<- interface{}, <-chan interface{}) {
	input := make(chan interface{})  // input channel for tasks
	output := make(chan interface{}) // output channel for tasks
	done := make(chan struct{})      // channel for workers to report that they are done

	for i := 0; i < numWorkers; i++ {
		// start worker Go routine
		go func(done chan<- struct{}) {
			for {
				//receive from input channel
				in, ok := <-input
				// if input channel empty: leave loop
				if !ok {
					break
				}
				// execute task and send result to output channel
				output <- task(in)
			}
			// report "done" to calling function
			done <- struct{}{}
		}(done)
	}

	// wait for all worker Go routine being done to close output channel
	go func() {
		for i := 0; i < numWorkers; i++ {
			<-done
		}
		close(output)
	}()

	return input, output
}
