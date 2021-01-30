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

// GetStep provides a method to read the absolute location of the stepper motor.
type GetStep interface {
	GetStep() int64
}

// Adjuster provides an interface to update the measured number of steps
// in a revolution of a hand.
type Adjuster interface {
	Adjust(int, int)
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
// An offset may be provided that correlates the encoder mark reference point
// and the actual physical location of the hand - when the hand is at the
// encoder reference point, the offset represents the relative offset of the
// physical clock hand e.g when the hand is at the encoder mark, the hand
// may be pointing to a location N steps away from the top of the clock face.
type Encoder struct {
	getStep  GetStep
	adjust   Adjuster
	enc      IO    // I/O from encoder hardware
	Invert   bool  // Invert input signal
	Measured int   // Measured steps per revolution
	size     int64 // Minimum span of sensor mark
	offset   int64 // Offset from encoder mark
	lastEdge int64 // Last location of encoder mark
}

// NewEncoder creates a new Encoder structure.
func NewEncoder(stepper GetStep, adj Adjuster, io IO, size, offset int) *Encoder {
	e := new(Encoder)
	e.getStep = stepper
	e.adjust = adj
	e.enc = io
	e.size = int64(size)
	e.offset = int64(offset)
	go e.driver()
	return e
}

// Location returns the current location as a relative position from the encoder mark
func (e *Encoder) Location() int {
	return int(e.getStep.GetStep() + e.offset - e.lastEdge)
}

// driver is the main goroutine for servicing the encoder.
// Edge triggered input values are read, and encoder marks are searched for.
// An encoder mark is a 0->1->0 transition of at least a minimum size, usually
// correlating to a physical sensor such as an interrupting shaft photo-sensor.
// The 1->0 transition is considered the reference point for measuring the
// number of steps in a revolution.
func (e *Encoder) driver() {
	last := int64(e.offset)
	e.lastEdge = int64(-1)
	lastMeasured := 0
	for {
		// Retrieve the sensor value when it changes.
		s, err := e.enc.Get()
		if err != nil {
			log.Fatalf("Encoder input: %v", err)
		}
		if e.Invert {
			s = s ^ 1
		}
		// Retrieve the current absolute location (offset included).
		loc := e.getStep.GetStep() + e.offset
		// Check for debounce, and discard if noisy.
		d := diff(loc, last)
		last = loc
		if debounce != 0 && d < debounce {
			continue
		}
		// Transitioned from 1 to 0, and the signal is large
		// enough to be considered as the real encoder mark.
		if s == 0 && d >= e.size {
			if e.lastEdge > 0 {
				// If the previous sensor edge has been seen,
				// calculate the difference between the current
				// mark and the previous mark.
				// This is the measured number of steps in a revolution.
				e.Measured = int(diff(e.lastEdge, loc))
				if lastMeasured != e.Measured {
					// If the number of steps in a revolution has
					// changed, update the interested party.
					log.Printf("Adjust to %d (%d)", e.Measured, e.Measured-lastMeasured)
					e.adjust.Adjust(e.Measured, int(e.offset))
					lastMeasured = e.Measured
				}
			}
			e.lastEdge = loc
		}
	}
}

// Get difference between 2 absolute locations.
func diff(a, b int64) int64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d
}
