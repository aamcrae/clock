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
	"time"
)

type MoveHand interface {
	Move(int)
}

// Hand represents a clock hand. A single revolution of the hand
// is represented by the number of ticks. Updating the clock
// will move the hand by one tick each time.
// Moving is done by sending a +/- step count to a mover.
// The number of steps in a single revolution is held in adjusted,
// which is initially set from a reference value, and can be
// updated by an external encoder tracking the actual physical
// movement of the hand.
type Hand struct {
	Name     string
	mover    MoveHand
	interval time.Duration
	ticks    int // Number of segments in clock face
	steps    int // Reference steps per clock revolution
	adjusted int // Measured steps per revolution
	divisor  int // Used to calculate ticks
	current  int // Current hand position
}

// NewHand creates and initialises a Hand structure.
func NewHand(name string, unit time.Duration, mover MoveHand, update time.Duration, steps int) *Hand {
	h := new(Hand)
	h.Name = name
	h.mover = mover
	h.interval = update
	h.ticks = int(unit / update)
	h.divisor = int(update.Milliseconds())
	h.steps = steps
	h.adjusted = steps
	fmt.Printf("%s: ticks %d, steps %d, divisor %d\n", h.Name, h.ticks, h.steps, h.divisor)
	return h
}

// Position returns the current hand position as well as the
// number of steps in a revolution.
func (h *Hand) Position() (int, int) {
	return h.current, h.adjusted
}

// Adjust sets an updated steps per revolution.
// An adjustment usually is derived from a sensor tracking the actual
// movememnt of the hand.
func (h *Hand) Adjust(adj int) {
	fmt.Printf("%s: New steps per revol = %d (old %d)\n", h.Name, adj, h.adjusted)
	h.adjusted = adj
}

// Run starts the processing of the hand. An initial value
// indicates the physical location of the hand, as steps around the
// clock face, and this is used to set the current location of the hand.
func (h *Hand) Run(initial int) {
	h.current = initial
	target := h.target(time.Now())
	fmt.Printf("%s: Setting initial position (%d steps)\n", h.Name, target-h.current)
	h.set(target)
	// Attempt to start ticker on the interval boundary
	h.syncTime()
	ticker := time.NewTicker(h.interval)
	fmt.Printf("%s: Interval %s, ticker started\n", h.Name, h.interval.String())
	for {
		h.set(h.target(<-ticker.C))
	}
}

// Set the hand to the target position
func (h *Hand) set(target int) {
	st := 0
	if target == 0 {
		st += h.adjusted - h.current
		h.current = 0
	} else {
		st += target - h.current
		h.current += st
	}
	h.mover.Move(st)
}

// Calculate and determine the target tick
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
	st := (mt*h.adjusted + h.ticks/2) / h.ticks
	return st
}

// Sync time to the boundary of the interval.
func (h *Hand) syncTime() {
	n := time.Now()
	adj := time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), time.UTC)
	tr := adj.Truncate(h.interval).Add(h.interval)
	time.Sleep(tr.Sub(adj))
}