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

package main

import (
	"github.com/aamcrae/clock/io"
)

type GetStep interface {
	Get() int64
}

type Adjuster interface {
	Adjust(int)
}

type IO interface {
	Wait() int
}

// Encoder is an interrupter encoder used to measure shaft rotations.
// The count of current step values is used to track the
// number of steps in a rotation between encoder signals, and
// this is used against the reference value to determine whether
// an adjustment should be made.
type Encoder struct {
	getStep   GetStep
	adjust    Adjuster
	enc		  IO		// I/O from encoder hardware
	reference int // Reference number of steps per revolution
	slots     int
}

// NewEncoder creates a new Encoder structure
func NewEncoder(stepper GetStep, adj Adjuster, io IO, reference, slots int) *Encoder {
	e := new(Encoder)
	e.getStep = stepper
	e.adjust = adj
	e.enc = IO
	e.slots = slots
	e.reference = reference
	go e.driver()
	return e
}

func (e *Encoder) driver() {
	// Poll input
	// track number of steps per rotation
	// Get count from stepper
	// figure out adjustment
}
