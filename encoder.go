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

// Encoder represents an input that changes when a shaft rotates.
// The count of current step values is used to track the
// number of steps in a rotation between encoder signals, and
// this is used to determine whether an adjustment should be made.
type Encoder struct {
	get    GetStep
	adjust Adjuster
	slots	int
	gpio   *io.Gpio
}

// NewEncoder creates a new Encoder structure
func NewEncoder(input *io.Gpio, stepper GetStep, adj Adjuster, slots int) (* Encoder){
	e := new(Encoder)
	e.gpio = input
	e.get = stepper
	e.adjust = adj
	e.slots = slots
	go e.driver()
	return e
}

func (e * Encoder) driver() {
	// Poll input
	// track number of steps per rotation
	// Get count from stepper
	// figure out adjustment
}
