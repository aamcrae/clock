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

// Interrupter encoder driver

package hand

import (
	"log"
)

type GetStep interface {
	GetStep() int64
}

type Adjuster interface {
	Adjust(int)
}

// Edge triggered input
type IO interface {
	Get() (int, error)
}

const debounce = 20

// Encoder is an interrupter encoder used to measure shaft rotations.
// The count of current step values is used to track the
// number of steps in a rotation between encoder signals, and
// this is used against the previous value to determine whether
// an adjustment should be made.
type Encoder struct {
	getStep  GetStep
	adjust   Adjuster
	enc      IO    // I/O from encoder hardware
	Invert   bool  // Invert input signal
	Measured int   // Measured steps per revolution
	size     int64 // Minimum size of sensor gap
	Midpoint int   // Midpoint of sensor.
}

// NewEncoder creates a new Encoder structure
func NewEncoder(stepper GetStep, adj Adjuster, io IO, size int) *Encoder {
	e := new(Encoder)
	e.getStep = stepper
	e.adjust = adj
	e.enc = io
	e.size = int64(size)
	go e.driver()
	return e
}

// Poll input
// track number of steps per rotation
// Get count from stepper
// figure out adjustment
func (e *Encoder) driver() {
	last := int64(-1)
	lastMid := -1
	start := int64(-1)
	for {
		// Sensor going high or low
		s, err := e.enc.Get()
		if err != nil {
			log.Fatalf("Encoder input: %v", err)
		}
		if e.Invert {
			s = s ^ 1
		}
		loc := e.getStep.GetStep()
		if last == -1 {
			last = loc
			continue
		}
		// Check for debounce
		d := diff(loc, last)
		last = loc
		if d < debounce {
			continue
		}
		if s == 1 {
			start = loc
		} else if d >= e.size {
			e.Midpoint = int((loc-start)/2 + start)
			if lastMid > 0 {
				// If the last sensor midpoint is known,
				// calculate the difference between the current
				// midpoint and the previous.
				// This is the measured number of steps in a revolution.
				e.Measured = int(diff(int64(lastMid), int64(e.Midpoint)))
				if lastMid != e.Measured {
					e.adjust.Adjust(e.Measured)
				}
			}
			lastMid = e.Midpoint
		}
	}
}

// Get absolute difference between 2 locations
func diff(a, b int64) int64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d
}
