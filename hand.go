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

package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

type MoveHand interface {
	Move(int)
}

// Hand represents a clock hand. A single revolution of the hand
// is represented by the number of ticks. Updating the clock
// will move the hand by one tick each time.
// Moving is done by sending a +/- step count to a mover.
// The total number of steps in a single revolution is held in steps.
// TODO: It's very likely that a single revolution is not an integral
// number of steps because of gearing, so it would be better to treat
// steps as a float, and keep a running accumulations of steps.
type Hand struct {
	name     string
	mover    MoveHand
	interval time.Duration
	ticks    int   // Number of segments in clock face
	steps    int   // Steps per clock revolution
	divisor  int   // Used to calculate ticks
	current  int   // Current hand position
	adjust   int32 // Adjustment to apply.
}

// NewHand creates and initialises a Hand structure.
func NewHand(name string, unit time.Duration, mover MoveHand, update time.Duration, steps int) *Hand {
	h := new(Hand)
	h.name = name
	h.mover = mover
	h.interval = update
	h.ticks = int(unit / update)
	h.divisor = int(update.Milliseconds())
	h.steps = steps
	fmt.Printf("%s: ticks %d, steps %d, divisor %d\n", h.name, h.ticks, h.steps, h.divisor)
	return h
}

func (h *Hand) Start(t time.Time) {
	h.current = h.target(t)
	go h.run()
}

// Adjust records an adjustment in half-steps that should be applied
// to the step counter on the next movement. An adjustment usually
// is derived from a sensor tracking the actual movememnt of the hand.
// A positive adjust will increase the number of steps on the next movement, which
// a negative value will reduce the number of steps.
func (h *Hand) Adjust(adj int) {
	atomic.AddInt32(&h.adjust, int32(adj))
}

func (h *Hand) run() {
	target := h.target(time.Now())
	fmt.Printf("%s: Setting initial position (%d steps)\n", h.name, target-h.current)
	h.set(target)
	// Attempt to start ticker on the interval boundary
	h.syncTime()
	ticker := time.NewTicker(h.interval)
	fmt.Printf("%s: Interval %s, ticker started\n", h.name, h.interval.String())
	for {
		h.set(h.target(<-ticker.C))
	}
}

// Set the hand to the target position
func (h *Hand) set(target int) {
	// Get the adjustment value
	st := int(atomic.SwapInt32(&h.adjust, 0))
	if target == 0 {
		st += h.steps - h.current
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
	st := mt * h.steps / h.ticks
	return st
}

// Sync time to the boundary of the interval.
func (h *Hand) syncTime() {
	n := time.Now()
	adj := time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), time.UTC)
	tr := adj.Truncate(h.interval).Add(h.interval)
	time.Sleep(tr.Sub(adj))
}
