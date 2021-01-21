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
	"github.com/aamcrae/config"
)

type StepperMover struct {
	name    string
	stepper *io.Stepper
	speed   float64
}

var startTime = flag.String("time", "3:04:05", "Current time on clock face")
var configFile = flag.String("config", "", "Configuration file")

func main() {
	flag.Parse()
	conf, err := config.ParseFile(*configFile)
	if err != nil {
		log.Fatalf("%s: %v", *configFile, err)
	}
	initial, err := time.Parse("3:04:05", *startTime)
	if err != nil {
		log.Fatalf("%s: %v", *startTime, err)
	}
	for _, sect := range []string{"hours", "minutes", "seconds"} {
		s := conf.GetSection(sect)
		if s != nil {
			setupHand(s, initial)
		}
	}
	select {}
}

// Sample config:
//  stepper=4,17,27,22,3.0
//  period=12h
//  update=5m
//  steps=4096
func setupHand(conf *config.Section, initial time.Time) {
	var speed float64
	var g [4]int
	n, err := conf.Parse("stepper", "%d,%d,%d,%d,%f", &g[0], &g[1], &g[2], &g[3], &speed)
	if err != nil || n != 5 {
		log.Fatalf("invalid stepper format: %v", err)
	}
	var gp[4] *io.Gpio
	for i, v := range g {
		gp[i], err = io.OutputPin(v)
		if err != nil {
			log.Fatalf("Pin %d: %v", v, err)
		}
	}
	var steps float64
	n, err = conf.Parse("steps", "%f", &steps)
	if err != nil || n != 1 {
		log.Fatalf("invalid steps: %v", err)
	}
	p, err := conf.GetArg("period")
	if err != nil {
		log.Fatalf("invalid period: %v", err)
	}
	period, err := time.ParseDuration(p)
	if err != nil {
		log.Fatalf("%s: invalid period: %v", p, err)
	}
	u, err := conf.GetArg("update")
	if err != nil {
		log.Fatalf("invalid update: %v", err)
	}
	update, err := time.ParseDuration(p)
	if err != nil {
		log.Fatalf("%s: invalid update: %v", u, err)
	}
	var encoder int
	n, err = conf.Parse("encoder", "%d", &encoder)
	if err != nil || n != 1 {
		encoder = 0
	}
	stepper := io.NewStepper(steps, gp[0], gp[1], gp[2], gp[3])
	mover := &StepperMover{conf.Name, stepper, speed}
	h := NewHand(conf.Name, period, mover, update, int(steps))
	h.Start(initial)
	if encoder != 0 {
		inp, err := io.Pin(encoder)
		if err != nil {
			log.Fatalf("Encoder %d: %v", encoder, err)
		}
		NewEncoder(inp, stepper, h, int(steps), 1)
	}
}

// Shim for move requests.
func (m *StepperMover) Move(steps int) {
	if m.stepper != nil {
		m.stepper.Step(m.speed, steps)
		m.stepper.Off() // Turn stepper off between moves
	}
}
