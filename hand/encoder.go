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

// Interrupter encoder driver.

package hand

import (
	"log"
)

// GetStep provides a method to read the current location of a hand.
type GetStep interface {
	GetStep() int64
}

// Adjuster provides an interface to update the measured number of steps
// in a revolution of a hand.
type Adjuster interface {
	Adjust(int)
}

// IO provides a method to return when an input changes.
type IO interface {
	Get() (int, error)
}

const debounce = 5

// Encoder is an interrupter encoder driver used to measure shaft rotations.
// The count of current step values is used to track the
// number of steps in a rotation between encoder signals, and
// this is used to calculate the actual number of steps in a revolution.
type Encoder struct {
	getStep  GetStep
	adjust   Adjuster
	enc      IO    // I/O from encoder hardware
	Invert   bool  // Invert input signal
	Measured int   // Measured steps per revolution
	size     int64 // Minimum span of sensor mark
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

// driver is the main goroutine for servicing the encoder.
func (e *Encoder) driver() {
	last := int64(0)
	lastEdge := int64(-1)
	lastMeasured := 0
	start := int64(-1)
	for {
		// Retrieve the sensor value when it changes.
		s, err := e.enc.Get()
		if err != nil {
			log.Fatalf("Encoder input: %v", err)
		}
		if e.Invert {
			s = s ^ 1
		}
		// Retrieve the current location.
		loc := e.getStep.GetStep()
		// Check for debounce
		d := diff(loc, last)
		last = loc
		if debounce != 0 && d < debounce {
			continue
		}
		// If transitioning from 0 to 1, remember this location as
		// the starting location of the encoder mark
		if s == 1 {
			start = loc
		} else if d >= e.size {
			// Transitioned from 1 to 0, and the signal is large
			// enough to be considered as the real encoder mark.
			if lastEdge > 0 {
				// If the last sensor edge is known,
				// calculate the difference between the current
				// edge and the previous.
				// This is the measured number of steps in a revolution.
				e.Measured = int(diff(lastEdge, loc))
				// Determine the midpoint of the encoder mark.
				e.Midpoint = int((loc-start)/2 + start) % e.Measured
				if lastMeasured != e.Measured {
					// If the number of steps in a revolution has
					// changed, update the interested party.
					e.adjust.Adjust(e.Measured)
					lastMeasured = e.Measured
				}
			}
			lastEdge = loc
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
