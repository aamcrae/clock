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
	"github.com/aamcrae/config"
)

type StepperMover struct {
	name    string
	stepper *io.Stepper
	speed   float64
}

type HandConfig struct {
	Name    string
	Gpio    []int
	Speed   float64
	Period  time.Duration
	Update  time.Duration
	Steps   float64
	Encoder int
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
			hc, err := handConfig(s)
			if err != nil {
				log.Fatalf("%s: %v", *configFile, err)
			}
			err = setupHand(hc, initial)
			if err != nil {
				log.Fatalf("%s: %v", hc.Name)
			}
		}
	}
	select {}
}

// Sample config:
//  stepper=4,17,27,22,3.0
//  period=12h
//  update=5m
//  steps=4096
func handConfig(conf *config.Section) (*HandConfig, error) {
	var err error
	var h HandConfig
	h.Name = conf.Name
	h.Gpio = make([]int, 4)
	n, err := conf.Parse("stepper", "%d,%d,%d,%d,%f", &h.Gpio[0], &h.Gpio[1], &h.Gpio[2], &h.Gpio[3], &h.Speed)
	if err != nil {
		return nil, fmt.Errorf("stepper: %v", err)
	}
	if n != 5 {
		return nil, fmt.Errorf("invalid stepper arguments")
	}
	n, err = conf.Parse("steps", "%f", &h.Steps)
	if err != nil {
		return nil, fmt.Errorf("steps: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("steps: argument count")
	}
	p, err := conf.GetArg("period")
	if err != nil {
		return nil, fmt.Errorf("period: %v", err)
	}
	h.Period, err = time.ParseDuration(p)
	if err != nil {
		return nil, fmt.Errorf("period: %v", err)
	}
	u, err := conf.GetArg("update")
	if err != nil {
		return nil, fmt.Errorf("update: %v", err)
	}
	h.Update, err = time.ParseDuration(u)
	if err != nil {
		return nil, fmt.Errorf("update: %v", err)
	}
	n, err = conf.Parse("encoder", "%d", &h.Encoder)
	if err != nil {
		return nil, fmt.Errorf("encoder: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("encoder: argument count")
	}
	return &h, nil
}

func setupHand(hc *HandConfig, initial time.Time) error {
	var gp [4]*io.Gpio
	var err error
	for i, v := range hc.Gpio {
		gp[i], err = io.OutputPin(v)
		if err != nil {
			return fmt.Errorf("Pin %d: %v", v, err)
		}
	}
	stepper := io.NewStepper(hc.Steps, gp[0], gp[1], gp[2], gp[3])
	mover := &StepperMover{hc.Name, stepper, hc.Speed}
	h := NewHand(hc.Name, hc.Period, mover, hc.Update, int(hc.Steps))
	inp, err := io.Pin(hc.Encoder)
	if err != nil {
		fmt.Errorf("Encoder %d: %v", hc.Encoder, err)
	}
	NewEncoder(stepper, h, inp, int(hc.Steps), 100)
	h.Start(initial)
	return nil
}

// Shim for move requests.
func (m *StepperMover) Move(steps int) {
	if m.stepper != nil {
		m.stepper.Step(m.speed, steps)
		m.stepper.Off() // Turn stepper off between moves
	}
}
