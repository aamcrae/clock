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
	"log"
	"time"

	"github.com/aamcrae/clock/io"
)

var gpios = []*int{
	flag.Int("m1", 4, "Minutes output 1"),
	flag.Int("m2", 17, "Minutes output 2"),
	flag.Int("m3", 27, "Minutes output 3"),
	flag.Int("m4", 22, "Minutes output 4"),
	flag.Int("h1", 6, "Hours output 1"),
	flag.Int("h2", 13, "Hours output 2"),
	flag.Int("h3", 19, "Hours output 3"),
	flag.Int("h4", 26, "Hours output 4"),
}
var encHours = flag.Int("hours_enc", 0, "Input for hours encoder")
var encMinutes = flag.Int("min_enc", 0, "Input for minutes encoder")
var encSeconds = flag.Int("sec_enc", 0, "Input for seconds encoder")

type StepperMover struct {
	name    string
	stepper *io.Stepper
	speed   float64
}

var startTime = flag.String("time", "3:04:05", "Current time on clock face")
var speed = flag.Float64("speed", 4.0, "Stepper speed in RPM")
var halfSteps = flag.Float64("steps", 2048*2, "Half steps in a revolution")

func main() {
	flag.Parse()
	initial, err := time.Parse("3:04:05", *startTime)
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
	setupHand("hours",
		io.NewStepper(*halfSteps, pins[4], pins[5], pins[6], pins[7]),
		time.Hour*12,
		time.Minute*5,
		*encHours,
		initial)

	setupHand("minutes",
		io.NewStepper(*halfSteps, pins[0], pins[1], pins[2], pins[3]),
		time.Hour,
		time.Second*10,
		*encMinutes,
		initial)

	setupHand("seconds",
		nil,
		time.Minute,
		time.Millisecond*250,
		*encSeconds,
		initial)

	select {}
}

func setupHand(name string, stepper *io.Stepper, period, update time.Duration, enc int, initial time.Time) {
	mover := &StepperMover{name, stepper, *speed}
	h := NewHand(name, period, mover, update, int(*halfSteps))
	h.Start(initial)
	if enc != 0 {
		inp, err := io.Pin(enc)
		if err != nil {
			log.Fatalf("Encoder %d: %v", enc, err)
		}
		NewEncoder(inp, stepper, h, int(*halfSteps), 1)
	}
}

// Shim for move requests.
func (m *StepperMover) Move(steps int) {
	if m.stepper != nil {
		m.stepper.Step(m.speed, steps)
		m.stepper.Off() // Turn stepper off between moves
	}
}
