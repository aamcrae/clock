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

// Clock program

package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aamcrae/clock/io"
)

const halfStepsRev = 2048 * 2

type hand struct {
	name     string
	stepper  *io.Stepper
	interval time.Duration
	ticks    int
	steps    int // Steps clock revolution
	divisor  int
	mod      int
	current  int
}

var gpios = []*int{
	flag.Int("a1", 4, "GPIO pin for motor output 1"),
	flag.Int("a2", 17, "GPIO pin for motor output 2"),
	flag.Int("a3", 27, "GPIO pin for motor output 3"),
	flag.Int("a4", 22, "GPIO pin for motor output 4"),
	flag.Int("b1", 6, "GPIO pin for motor B input 1"),
	flag.Int("b2", 13, "GPIO pin for motor B input 2"),
	flag.Int("b3", 19, "GPIO pin for motor B input 3"),
	flag.Int("b4", 26, "GPIO pin for motor B input 4"),
}

var startTime = flag.String("time", "3:04:05", "Current time on clock face")
var speed = flag.Float64("speed", 4.0, "Stepper speed in RPM")

func main() {
	flag.Parse()
	face, err := time.Parse("3:04:05", *startTime)
	if err != nil {
		log.Fatalf("%s: %v", *startTime, err)
	}
	pins := make([]*io.Gpio, len(gpios))
	for i, gp := range gpios {
		var err error
		pins[i], err = io.OutputPin(*gp)
		if err != nil {
			log.Fatalf("Pin %d: %v", *gp, err)
		}
		defer pins[i].Close()
	}
	hour := new(hand)
	hour.name = "hours"
	hour.stepper = io.NewStepper(halfStepsRev, pins[4], pins[5], pins[6], pins[7])
	hour.interval = time.Minute * 5
	hour.ticks = 12 * 12
	hour.divisor = 60 * 5 * 1000
	hour.steps = halfStepsRev
	hour.current = hour.target(face)

	min := new(hand)
	min.name = "minutes"
	min.stepper = io.NewStepper(halfStepsRev, pins[0], pins[1], pins[2], pins[3])
	min.interval = time.Second * 10
	min.ticks = 60 * 6
	min.divisor = 10 * 1000
	min.steps = halfStepsRev
	min.current = min.target(face)

	sec := new(hand)
	sec.name = "seconds"
	sec.interval = time.Second
	sec.ticks = 60
	sec.divisor = 1000
	sec.steps = halfStepsRev
	sec.current = sec.target(face)

	go hour.run()
	go min.run()
	// go sec.run()
	select {}
}

func (h *hand) run() {
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
func (h *hand) set(target int) {
	var st int
	if target == 0 {
		st = h.steps - h.current
		h.current = 0
	} else {
		st = target - h.current
	}
	h.current += st
	if h.stepper != nil {
		h.stepper.Step(*speed, st)
		h.stepper.Off()
	} else {
		fmt.Printf("%s: step %d\n", h.name, st)
	}
}

// Calculate and determine the target tick
func (h *hand) target(t time.Time) int {
	// Calculate milliseconds of day.
	hour, minute, sec := t.Clock()
	target := (hour % 12) * 60 * 60 * 1000
	target += minute * 60 * 1000
	target += sec * 1000
	target += t.Nanosecond() / 1_000_000
	return ((target / h.divisor) % h.ticks) * h.steps / h.ticks
}

// Sync time to the boundary of the interval.
func (h *hand) syncTime() {
	n := time.Now()
	adj := time.Date(n.Year(), n.Month(), n.Day(), n.Hour(), n.Minute(), n.Second(), n.Nanosecond(), time.UTC)
	tr := adj.Truncate(h.interval).Add(h.interval)
	time.Sleep(tr.Sub(adj))
}
