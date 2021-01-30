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

// Clock hand processing

package hand

import (
	"fmt"
	"sync"
	"time"
)

// MoveHand is the interface to move the clock hand.
type MoveHand interface {
	Move(int)
}

// Hand represents a clock hand. A single revolution of the hand
// is represented by the number of ticks. Updating the clock
// moves the hand by one tick each time.
// Moving is done by sending a +/- step count to a mover.
// The number of steps in a single revolution is held in actual,
// which is initially set from a reference value, and can be
// updated by an external encoder tracking the actual physical
// movement of the hand.
type Hand struct {
	Name      string
	Ticking   bool
	Current   int // Current hand position
	Adjusted  int // Number of times adjustment has been made
	mover     MoveHand
	interval  time.Duration
	ticks     int        // Number of segments in clock face
	reference int        // Reference steps per clock revolution
	actual    int        // Measured steps per revolution
	divisor   int        // Used to calculate ticks
	mu        sync.Mutex // Guards Current and actual
}

// NewHand creates and initialises a Hand structure.
func NewHand(name string, unit time.Duration, mover MoveHand, update time.Duration, steps int) *Hand {
	h := new(Hand)
	h.Name = name
	h.mover = mover
	h.interval = update
	h.ticks = int(unit / update)
	h.divisor = int(update.Milliseconds())
	h.reference = steps
	h.actual = steps // Initial reference value
	h.Current = 0
	fmt.Printf("%s: ticks %d, reference steps %d, divisor %d\n", h.Name, h.ticks, h.reference, h.divisor)
	return h
}

// Position returns the current hand position as well as the
// number of steps in a revolution.
func (h *Hand) Position() (int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Current, h.actual
}

// Adjust sets an updated steps per revolution and resets the current location.
// An adjustment usually is derived from a sensor tracking the actual
// movemment of the hand.
func (h *Hand) Adjust(adj int, location int) {
	h.Adjusted++
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actual = adj
	h.Current = location
}

// Run starts the processing of the hand.
// The hand processing basically involves starting a ticker at the update
// rate specified for the hand, and then moving the hand to match the time
// the ticker sends.
func (h *Hand) Run() {
	target := h.target(time.Now())
	h.set(target)
	// Attempt to start ticker on the update boundary so that the ticker
	// ticks on the precise time of the update interval
	h.syncTime()
	ticker := time.NewTicker(h.interval)
	h.Ticking = true
	for {
		// Receive the time from the ticker, and set the hand to the
		// target position matching the time.
		h.set(h.target(<-ticker.C))
	}
}

// Set the hand to the target position.
// Always move clockwise, to avoid encoder getting confused.
func (h *Hand) set(target int) {
	st := 0
	h.mu.Lock()
	st = target - h.Current
	if st < 0 {
		st += h.actual
	}
	h.Current = (st + h.Current) % h.actual
	h.mu.Unlock()
	if st > 0 {
		h.mover.Move(st)
	}
}

// Calculate and determine the target position of the hand
// given the time and the current parameters of the hand (i.e
// the number of steps in a revolution of the hand.
func (h *Hand) target(t time.Time) int {
	// Calculate milliseconds of day.
	hour, minute, sec := t.Clock()
	target := (hour % 12) * 60 * 60 * 1000
	target += minute * 60 * 1000
	target += sec * 1000
	target += t.Nanosecond() / 1_000_000
	mod := (target / h.divisor)
	mt := mod % h.ticks
	// Round up.
	return (mt*h.actual + h.ticks/2) / h.ticks
}

// Sync time to the boundary of the update interval so that the
// ticker is aligned to the update time e.g if the update interval
// of a hand is 10 seconds, then make sure the ticker is sending a tick
// at 0, 10, 20 seconds (rather than 1, 11, 21...)
func (h *Hand) syncTime() {
	n := time.Now()
	adj := time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), time.UTC)
	tr := adj.Truncate(h.interval).Add(h.interval)
	time.Sleep(tr.Sub(adj))
}
