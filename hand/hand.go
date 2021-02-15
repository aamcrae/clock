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
	"log"
	"sync"
	"time"
)

// MoveHand is the interface to move the hand by a selected number of steps.
type MoveHand interface {
	Move(int)
	GetLocation() int64 // Get current location
}

// Hand represents a clock hand. A single revolution of the hand
// is represented by a number of ticks, determined by the update duration
// for the hand e.g a minute hand takes 60 minutes for a revolution, and is
// updated every 5 seconds (to make the motion smooth), so this hand has 720 ticks (60 * 60 / 5).
// Ticking the clock involves moving the hand by one tick each update period.
// Ticks are not steps; a single tick is usually a number of steps.
//
// The number of steps in a single revolution is held in actual,
// which is initially set from a reference value, and can be
// updated by an external encoder tracking the actual physical
// movement of the hand.
//
// Moving is done by calculating the step location of a tick, and then
// sending a step count to a Mover to move the hand to the required step location.
// E.g if there are 4000 steps in a revolution, and 720 ticks for a minute hand,
// and the time is 17 minutes past the hour, this is tick 720 * 17 / 60 = 204.
// The target step location for this tick is 4000 * 204 / 720 = 1133.
// The current step number is used to determine how many steps the hand
// needs to move to get to 1133.
//
// Moving is only performed in a clockwise direction.
// An offset may be provided that correlates the encoder mark reference point
// and the actual physical location of the hand - when the hand is at the
// encoder reference point, the offset represents the relative offset of the
// physical clock hand e.g when the hand is at the encoder mark, the offset represents
// the location of the hand as steps away from the top of the clock face.
type Hand struct {
	Name        string        // Name of this hand
	Ticking     bool          // True if the clock has completed initialisation and is ticking.
	base        int64         // position of last encoder mark
	mover       MoveHand      // Mover to move the hand
	update      time.Duration // Update interval
	ticks       int           // Number of segments in clock face
	reference   int           // Reference steps per clock revolution
	actual      int           // Measured steps per revolution
	divisor     int           // Used to calculate ticks
	skipMove    int           // Minimum amount required to fast forward
	offset      int           // Offset of hand at encoder mark
	mu          sync.Mutex    // Guards base and actual
	Marks       int           // Number of times encoder mark hit
	Skipped     int           // Number of skipped moves
	FastForward int           // Number of fast forward movements
	Adjusted    int           // Number of hand adjustments
}

// NewHand creates and initialises a Hand structure.
func NewHand(name string, unit time.Duration, mover MoveHand, update time.Duration, steps, offset int) *Hand {
	h := new(Hand)
	h.Name = name
	h.mover = mover
	h.update = update
	h.ticks = int(unit / update)
	h.divisor = int(update.Milliseconds())
	h.reference = steps
	h.actual = steps // Initial reference value
	h.offset = offset
	h.skipMove = steps / 100
	log.Printf("%s: ticks %d, reference steps %d, divisor %d, offset %d\n", h.Name, h.ticks, h.reference, h.divisor, h.offset)
	return h
}

// Get returns the current relative position,
// the number of steps in a revolution, and
// the current offset.
func (h *Hand) Get() (int, int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.getCurrent(), h.actual, h.offset
}

// Adjust adjusts the offset so that the physical position can be tweaked.
// A positive value reduces the offset so that the hand is closer to the
// encoder mark. The initial offset in the configuration should also
// be adjusted.
func (h *Hand) Adjust(adj int) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Adjusted++
	h.offset -= adj
	if h.offset < 0 {
		h.offset += h.actual
	} else {
		h.offset %= h.actual
	}
	return h.offset
}

// Mark updates the steps per revolution and sets the current location to a preset value.
// Usually called from a sensor encoder at the point when an encoder mark is detected, indicating
// a known physical location of the hand.
func (h *Hand) Mark(adj int, loc int64) {
	h.Marks++
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actual = adj
	// Reset the current location.
	h.base = loc
}

// Calculate the current location of the hand.
func (h *Hand) getCurrent() int {
	return (int(h.mover.GetLocation()-h.base) + h.offset) % h.actual
}

// Run starts the ticking of the hand.
// The hand processing basically involves starting a ticker at the update
// rate specified for the hand, and then moving the hand to match the step location
// correlating to the time value the ticker sends.
func (h *Hand) Run() {
	// Get the step location corresponding to the current time.
	target := h.target(time.Now())
	// Move the hand to the target location.
	log.Printf("%s: Initial target %d, current %d", h.Name, target, h.getCurrent())
	h.moveTo(target)
	// Attempt to start a Ticker on the update boundary so that the ticker
	// ticks as close as possible on the exact time of the update interval.
	h.syncTime()
	ticker := time.NewTicker(h.update)
	h.Ticking = true
	for {
		// Receive the time from the ticker, and set the hand to the
		// target position calculated from the current time.
		h.moveTo(h.target(<-ticker.C))
	}
}

// Set the hand to the target position.
// Always move clockwise, to avoid encoder getting confused.
func (h *Hand) moveTo(target int) {
	st := h.steps(target)
	if st > 0 {
		h.mover.Move(st)
	}
}

// steps returns the number of steps to move
// from the current position to the target position.
// A small negative movement can arise if the steps per revolution is adjusted and the current location
// is now slightly ahead of where it should be.
// It is likely better to simply pause the hand to let time catch up;
// the alternative is to fast-forward the hand to the target point.
func (h *Hand) steps(target int) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Get difference between target and current location.
	cur := h.getCurrent()
	st := target - cur
	if st < 0 {
		if -st < h.skipMove {
			h.Skipped++
			log.Printf("%s: Skipping move (%d steps, %d current, %d target, %d actual)", h.Name, st, cur, target, h.actual)
			return 0
		}
		// Convert backwards move to forward move around the clock.
		st += h.actual
	}
	if st > h.skipMove {
		h.FastForward++
		log.Printf("%s: Fast foward (%d steps, %d current, %d target, %d actual, %d base)", h.Name, st, cur, target, h.actual, h.base)
	}
	return st
}

// Calculate and determine the target step position of the hand
// given the time and the current parameters of the hand (i.e
// the measured number of steps in a revolution of the hand).
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

// syncTime sleeps so that when the update interval Ticker is started, the
// Ticker is aligned to the update time e.g if the update interval
// of a hand is 10 seconds, then make sure the ticker is sending a tick
// at 0, 10, 20 seconds (rather than 1, 11, 21...).
func (h *Hand) syncTime() {
	n := time.Now()
	adj := time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), time.UTC)
	tr := adj.Truncate(h.update).Add(h.update)
	time.Sleep(tr.Sub(adj))
}
