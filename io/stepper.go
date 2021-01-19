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

type msg struct {
	speed float64 // RPM
	steps int
	sync  chan bool
}

// Stepper represents one stepper motor
type Stepper struct {
	g1, g2, g3, g4 *Gpio
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
func NewStepper(steps, g1, g2, g3, g4 int) (*Stepper, error) {
	if steps <= 30 {
		return nil, fmt.Errorf("invalid steps per revolution")
	}
	s := new(Stepper)
	s.factor = float64(time.Second.Nanoseconds()*60) / float64(steps*2)
	var err error
	s.g1, err = OutputPin(g1)
	if err != nil {
		return nil, err
	}
	s.g2, err = OutputPin(g2)
	if err != nil {
		s.Close()
		return nil, err
	}
	s.g3, err = OutputPin(g3)
	if err != nil {
		s.Close()
		return nil, err
	}
	s.g4, err = OutputPin(g4)
	if err != nil {
		s.Close()
		return nil, err
	}
	s.mChan = make(chan msg, 20)
	s.stopChan = make(chan bool)
	go s.handler()
	return s, nil
}

// Close disables the outputs and frees resources
func (s *Stepper) Close() {
	if s.g1 != nil {
		s.g1.Close()
	}
	if s.g2 != nil {
		s.g2.Close()
	}
	if s.g3 != nil {
		s.g3.Close()
	}
	if s.g4 != nil {
		s.g4.Close()
	}
	if s.mChan != nil {
		s.stopChan <- true
	}
}

// Step runs the stepper motor for number of steps at the RPM selected.
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
					s.die()
					return
				}
			}
			if m.sync != nil {
				m.sync <- true
				close(m.sync)
			}
		case <-s.stopChan:
			s.die()
			return
		}
	}
}

func (s *Stepper) step(rpm float64, steps int) bool {
	inc := 1
	if steps < 0 {
		inc = -1
		steps = -steps
	}
	steps = steps * 2
	delay := time.Duration(s.factor / rpm)
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	for i := 0; i < steps; i++ {
		seq := sequence[s.index]
		s.index = (s.index + inc) & 7
		s.g1.Set(seq[0])
		s.g2.Set(seq[1])
		s.g3.Set(seq[2])
		s.g4.Set(seq[3])
		select {
		case <-s.stopChan:
			return true
		case <-ticker.C:
		}
	}
	return false
}

// Handler is stopping.
func (s *Stepper) die() {
	close(s.mChan)
	close(s.stopChan)
}
