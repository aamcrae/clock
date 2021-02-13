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

// Syncer provides an interface for a callback when the encoder mark is hit.
// The measured steps in a revolution is provided.
type Syncer interface {
	Mark(int)
}

// IO provides a method to return when an input changes.
type IO interface {
	Get() (int, error)
}

const debounce = 5
const mAvgCount = 5

// Encoder is an interrupter encoder driver used to measure shaft rotations.
// The count of current step values is used to track the
// number of steps in a rotation between encoder signals, and
// this is used to calculate the actual number of steps in a revolution.
type Encoder struct {
	Name     string
	getStep  GetStep
	syncer   Syncer
	enc      IO    // I/O from encoder hardware
	Invert   bool  // Invert input signal
	Measured int   // Measured steps per revolution
	size     int64 // Minimum span of sensor mark
	lastEdge int64 // Last location of encoder mark
}

// NewEncoder creates a new Encoder structure.
func NewEncoder(name string, stepper GetStep, syncer Syncer, io IO, size int) *Encoder {
	e := new(Encoder)
	e.Name = name
	e.getStep = stepper
	e.syncer = syncer
	e.enc = io
	e.size = int64(size)
	go e.driver()
	return e
}

// Location returns the current location as a relative position from the encoder mark
func (e *Encoder) Location() int {
	return int(e.getStep.GetStep() - e.lastEdge)
}

// driver is the main goroutine for servicing the encoder.
// Edge triggered input values are read, and encoder marks are searched for.
// An encoder mark is a 0->1->0 transition of at least a minimum size, usually
// correlating to a physical sensor such as an interrupting shaft photo-sensor.
// The 1->0 transition is considered the reference point for measuring the
// number of steps in a revolution.
func (e *Encoder) driver() {
	last := int64(0)
	e.lastEdge = int64(-1)
	lastMeasured := 0
	var mavg []int
	avgTotal := 0
	avgIndex := 0
	for {
		// Retrieve the sensor value when it changes.
		s, err := e.enc.Get()
		if err != nil {
			log.Fatalf("%s: Encoder input: %v", e.Name, err)
		}
		if e.Invert {
			s = s ^ 1
		}
		// Retrieve the current absolute location.
		loc := e.getStep.GetStep()
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
				newM := int(diff(e.lastEdge, loc))
				if avgTotal == 0 {
					// If first time, init moving average.
					for i := 0; i < mAvgCount; i++ {
						mavg = append(mavg, newM)
					}
					avgTotal = newM * mAvgCount
				}
				// Recalculate moving average.
				avgTotal = avgTotal - mavg[avgIndex] + newM
				avgIndex = (avgIndex + 1) % mAvgCount
				newM = avgTotal / mAvgCount
				e.Measured = newM
				e.syncer.Mark(newM)
				log.Printf("%s: Mark at %d (%d)", e.Name, e.Measured, e.Measured-lastMeasured)
				lastMeasured = newM
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
