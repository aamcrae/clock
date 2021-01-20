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
	"fmt"
	"time"
)

type Setter interface {
	Set(int) error
}

type msg struct {
	speed float64 // RPM
	steps int
	sync  chan bool
}

// Stepper represents one stepper motor
type Stepper struct {
	pin1, pin2, pin3, pin4 Setter
	factor         float64
	mChan          chan msg
	stopChan       chan bool
	index          int
}

// Half step sequence
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

// NewStepper creates and initialises a Stepper
// halfSteps is the number of half steps per revolution.
func NewStepper(halfSteps int, pin1, pin2, pin3, pin4 Setter) (*Stepper, error) {
	if halfSteps <= 30 {
		return nil, fmt.Errorf("invalid steps per revolution")
	}
	s := new(Stepper)
	s.factor = float64(time.Second.Nanoseconds()*60) / float64(halfSteps)
	s.pin1 = pin1
	s.pin2 = pin2
	s.pin3 = pin3
	s.pin4 = pin4
	s.mChan = make(chan msg, 20)
	s.stopChan = make(chan bool)
	go s.handler()
	return s, nil
}

// Close disables the outputs and frees resources
func (s *Stepper) Close() {
	if s.mChan != nil {
		s.Stop()
		close(s.mChan)
		close(s.stopChan)
	}
}

// Save returns the current sequence index.
func (s *Stepper) Save() int {
	return s.index
}

// Restore initialises the sequence index to this value and
// initialises the outputs
func (s *Stepper) Restore(i int) {
	s.index = i & 7
	s.output()
}

// Stop aborts any current stepping.
func (s *Stepper) Stop() {
	s.stopChan <- true
	s.Wait()
}

// Step runs the stepper motor for number of half steps at the RPM selected.
// If steps is positive, then the motor is run clockwise, otherwise ccw.
func (s *Stepper) Step(rpm float64, steps int) {
	if steps != 0 && rpm > 0.0 {
		s.mChan <- msg{speed: rpm, steps: steps}
	}
}

// Wait waits for all commands to complete
func (s *Stepper) Wait() {
	c := make(chan bool)
	s.mChan <- msg{speed: 0, steps: 0, sync: c}
	<-c
}

// Background handler.
// Listens on message channel, and controls the motor.
func (s *Stepper) handler() {
	for {
		select {
		case m := <-s.mChan:
			if m.steps != 0 {
				if s.step(m.speed, m.steps) {
					return
				}
			}
			if m.sync != nil {
				m.sync <- true
				close(m.sync)
			}
		case stop := <-s.stopChan:
			s.flush()
			if !stop {
				return
			}
		}
	}
}

func (s *Stepper) step(rpm float64, steps int) bool {
	inc := 1
	if steps < 0 {
		inc = -1
		steps = -steps
	}
	delay := time.Duration(s.factor / rpm)
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for i := 0; i < steps; i++ {
		s.index = (s.index + inc) & 7
		s.output()
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
				// Channel has been closed.
				return
			}
		default:
			return
		}
	}
}

// Set the GPIO outputs
func (s *Stepper) output() {
	seq := sequence[s.index]
	s.pin1.Set(seq[0])
	s.pin2.Set(seq[1])
	s.pin3.Set(seq[2])
	s.pin4.Set(seq[3])
}
