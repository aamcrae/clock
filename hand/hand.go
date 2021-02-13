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

// MoveHand is the interface to move the clock hand.
type MoveHand interface {
	Move(int)
}

// Hand represents a clock hand. A single revolution of the hand
// is represented by a number of ticks, determined by the update duration
// for the hand.
// Ticking the clock involves moving the hand by one tick each update period.
// Moving is done by sending a step count to a Mover.
// Moving is only performed in a clockwise direction.
// The number of steps in a single revolution is held in actual,
// which is initially set from a reference value, and can be
// updated by an external encoder tracking the actual physical
// movement of the hand.
// An offset may be provided that correlates the encoder mark reference point
// and the actual physical location of the hand - when the hand is at the
// encoder reference point, the offset represents the relative offset of the
// physical clock hand e.g when the hand is at the encoder mark, the offset represents
// the location of the hand as steps away from the top of the clock face.
type Hand struct {
	Name        string        // Name of this hand
	Ticking     bool          // True if the clock has completed initialisation and is ticking.
	Current     int           // Current hand position
	mover       MoveHand      // Mover to move the hand
	update      time.Duration // Update interval
	ticks       int           // Number of segments in clock face
	reference   int           // Reference steps per clock revolution
	actual      int           // Measured steps per revolution
	divisor     int           // Used to calculate ticks
	skipMove    int           // Minimum amount required to fast forward
	offset      int           // Offset of hand at encoder mark
	mu          sync.Mutex    // Guards Current and actual
	Marks       int           // Number of times encoder mark hit
	Skipped     int           // Number of skipped moves
	FastForward int           // Number of fast forward movements
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
	h.Current = 0
	h.skipMove = steps / 100
	log.Printf("%s: ticks %d, reference steps %d, divisor %d\n", h.Name, h.ticks, h.reference, h.divisor)
	return h
}

// Set sets the current location of the hand, as measured from the
// encoder mark.
func (h *Hand) Set(pos int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Current = (pos + h.offset) % h.actual
	log.Printf("%s: Setting hand to encoder offset %d (location %d)", h.Name, pos, h.Current)
}

// Get returns the current relative position as well as the
// number of steps in a revolution.
func (h *Hand) Get() (int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Current, h.actual
}

// Adjust adjusts the offset so that the physical location can be tweaked.
func (h *Hand) Adjust(adj int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.offset = (h.offset + adj) % h.actual
	// Also apply the adjustment to the current location.
	h.Current = (h.Current + adj) % h.actual
}

// Mark updates the steps per revolution and sets the current location to a known point.
// An adjustment usually is derived from a sensor tracking the physical
// movement of the hand.
func (h *Hand) Mark(adj int) {
	h.Marks++
	h.mu.Lock()
	defer h.mu.Unlock()
	h.actual = adj
	// Reset the current location.
	h.Current = h.offset
}

// Run starts the ticking of the hand.
// The hand processing basically involves starting a ticker at the update
// rate specified for the hand, and then moving the hand to match the time
// the ticker sends.
func (h *Hand) Run() {
	// Move the hand to the position representing the current time.
	target := h.target(time.Now())
	h.moveTo(target)
	// Attempt to start a Ticker on the update boundary so that the ticker
	// ticks on the time of the update interval.
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

// steps returns the number of steps to move.
// A small negative movement can arise if the steps per revolution change.
// It is likely better to simply skip moving the hand to let the adjusted
// time catch up; the alternative is to fast-forward the hand to the target point.
func (h *Hand) steps(target int) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Get difference between target and current location.
	st := target - h.Current
	if st < 0 {
		if -st < h.skipMove {
			h.Skipped++
			log.Printf("%s: Skipping move (%d steps, %d current, %d target, %d actual)", h.Name, st, h.Current, target, h.actual)
			return 0
		}
		// Convert backwards move to forward move around the clock.
		st += h.actual
	}
	if st > h.skipMove {
		h.FastForward++
		log.Printf("%s: Fast foward (%d steps, %d current, %d target, %d actual)", h.Name, st, h.Current, target, h.actual)
	}
	// Update the current location to where the hand will be after the movement.
	h.Current = (st + h.Current) % h.actual
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
