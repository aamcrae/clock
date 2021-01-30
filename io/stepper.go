// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package io

import (
	"sync/atomic"
	"time"
)

const stepperQueueSize = 20 // Size of queue for requests

type msg struct {
	speed float64 // RPM
	steps int
	sync  chan bool
}

// Stepper represents a stepper motor.
// All actual stepping is done in a background goroutine, so requests can be queued.
// All step values assume half-steps.
// The current step number is maintained as an absolute number, referenced from
// 0 when the stepper is first initialised. This can be a negative or positive number,
// depending on the movement.
type Stepper struct {
	pin1, pin2, pin3, pin4 Setter    // Pins for controlling outputs
	factor                 float64   // Number of steps per revolution.
	mChan                  chan msg  // channel for message requests
	stopChan               chan bool // channel for signalling resets.
	index                  int       // Index to step sequence
	on                     bool      // true if motor drivers on
	current                int64     // Current step number as an absolute number
}

// Half step sequence of outputs.
var sequence = [][]int{
	[]int{1, 0, 0, 0},
	[]int{1, 1, 0, 0},
	[]int{0, 1, 0, 0},
	[]int{0, 1, 1, 0},
	[]int{0, 0, 1, 0},
	[]int{0, 0, 1, 1},
	[]int{0, 0, 0, 1},
	[]int{1, 0, 0, 1},
}

// NewStepper creates and initialises a Stepper struct, representing
// a stepper motor controlled by 4 GPIO pins.
// rev is the number of steps per revolution as a reference value for
// determining the delays between steps.
func NewStepper(rev int, pin1, pin2, pin3, pin4 Setter) *Stepper {
	s := new(Stepper)
	// Precalculate a timing factor so that a RPM value can be used
	// to calculate the per-sequence step delay.
	s.factor = float64(time.Second.Nanoseconds()*60) / float64(rev)
	s.pin1 = pin1
	s.pin2 = pin2
	s.pin3 = pin3
	s.pin4 = pin4
	s.mChan = make(chan msg, stepperQueueSize)
	s.stopChan = make(chan bool)
	go s.handler()
	return s
}

// Close stops the motor and frees any resources.
func (s *Stepper) Close() {
	s.Stop()
	close(s.mChan)
	close(s.stopChan)
}

// State returns the current sequence index, so that the current state
// of the motor can be saved and then restored in a new instance.
// This allows the exact state of the motor to be restored
// across process restarts so that the maximum accuracy can be guaranteed.
func (s *Stepper) State() int {
	return s.index
}

// GetStep returns the current step number, which is an accumulative
// signed value representing the steps moved, with 0 as the starting location.
func (s *Stepper) GetStep() int64 {
	return atomic.LoadInt64(&s.current)
}

// Restore initialises the sequence index to this value and sets the outputs
func (s *Stepper) Restore(i int) {
	s.index = i & 7
}

// Off turns off the GPIOs to remove the power from the motor.
func (s *Stepper) Off() {
	if s.on {
		s.Wait()
		s.pin1.Set(0)
		s.pin2.Set(0)
		s.pin3.Set(0)
		s.pin4.Set(0)
		s.on = false
	}
}

// Stop aborts any current stepping, and flushes all queued requests.
func (s *Stepper) Stop() {
	s.stopChan <- true
	s.Wait()
}

// Step queues a request to step the motor at the RPM selected for the
// number of half-steps.
// If halfSteps is positive, then the motor is run clockwise, otherwise ccw.
// A number of requests can be queued.
func (s *Stepper) Step(rpm float64, halfSteps int) {
	if halfSteps != 0 && rpm > 0.0 {
		if !s.on {
			s.output()
			s.on = true
		}
		s.mChan <- msg{speed: rpm, steps: halfSteps}
	}
}

// Wait waits for all requests to complete
func (s *Stepper) Wait() {
	c := make(chan bool)
	s.mChan <- msg{speed: 0, steps: 0, sync: c}
	<-c
}

// goroutine handler
// Listens on message channel, and runs the motor.
func (s *Stepper) handler() {
	for {
		select {
		case m := <-s.mChan:
			// Request to step the motor
			if m.steps != 0 {
				if s.step(m.speed, m.steps) {
					return
				}
			}
			if m.sync != nil {
				// If sync channel is present, signal it.
				m.sync <- true
				close(m.sync)
			}
		case stop := <-s.stopChan:
			// Request to stop and flush all requests
			s.flush()
			if !stop {
				return
			}
		}
	}
}

// step controls the motor via the GPIOs, to move the motor the
// requested number of steps. A negative value moves the motor
// counter-clockwise, positive moves the motor clockwise.
// Once started, a stop channel is used to abort the sequence.
func (s *Stepper) step(rpm float64, steps int) bool {
	inc := 1
	if steps < 0 {
		// Counter-clockwise
		inc = -1
		steps = -steps
	}
	// Calculate the per-step delay in nanoseconds by using the timing factor
	// and requested RPM, and use a ticker to signal the step sequence.
	delay := time.Duration(s.factor / rpm)
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for i := 0; i < steps; i++ {
		s.index = (s.index + inc) & 7
		s.output()
		atomic.AddInt64(&s.current, int64(inc))
		select {
		case stop := <-s.stopChan:
			s.flush()
			if stop {
				// Abort current stepping loop
				return false
			} else {
				// channel is closed, so kill handler.
				return true
			}
		case <-ticker.C:
		}
	}
	return false
}

// Flush all remaining actions from message channel.
func (s *Stepper) flush() {
	for {
		select {
		case m := <-s.mChan:
			if m.sync != nil {
				m.sync <- true
				close(m.sync)
			} else if m.steps == 0 && m.speed == 0.0 {
				// nil msg, channel has been closed.
				return
			}
		default:
			return
		}
	}
}

// Set the GPIO outputs according to the current sequence index.
func (s *Stepper) output() {
	seq := sequence[s.index]
	s.pin1.Set(seq[0])
	s.pin2.Set(seq[1])
	s.pin3.Set(seq[2])
	s.pin4.Set(seq[3])
}
