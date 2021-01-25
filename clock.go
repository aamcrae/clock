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

	"github.com/aamcrae/clock/hand"
	"github.com/aamcrae/clock/io"
	"github.com/aamcrae/config"
)

// StepperMover is a shim interface between
// the hand and the stepper motor.
type StepperMover struct {
	name    string
	stepper *io.Stepper
	speed   float64
}

// Configuration data for the hand.
type ClockConfig struct {
	Name    string
	Gpio    []int
	Speed   float64
	Period  time.Duration
	Update  time.Duration
	Steps   int
	Encoder int
	Initial int
}

type Clock struct {
	h    *hand.Hand
	enc  *hand.Encoder
	conf *ClockConfig
}

var configFile = flag.String("config", "", "Configuration file")
var port = flag.Int("port", 8080, "Web server port number")

func main() {
	flag.Parse()
	conf, err := config.ParseFile(*configFile)
	if err != nil {
		log.Fatalf("%s: %v", *configFile, err)
	}
	var clock []*Clock
	for _, sect := range []string{"hours", "minutes", "seconds"} {
		s := conf.GetSection(sect)
		if s != nil {
			hc, err := handConfig(s)
			if err != nil {
				log.Fatalf("%s: %v", *configFile, err)
			}
			c, err := setup(hc)
			if err != nil {
				log.Fatalf("%s: %v", hc.Name)
			}
			clock = append(clock, c)
		}
	}
	if *port != 0 {
		var hands []*hand.Hand
		for _, c := range clock {
			hands = append(hands, c.h)
		}
		hand.ClockServer(*port, hands)
	}
	select {}
}

// handConfig reads a hand config from a config file section.
// Sample config:
//  stepper=4,17,27,22,3.0
//  period=12h
//  update=5m
//  steps=4096
//  initial=2100
func handConfig(conf *config.Section) (*ClockConfig, error) {
	var err error
	var h ClockConfig
	h.Name = conf.Name
	h.Gpio = make([]int, 4)
	n, err := conf.Parse("stepper", "%d,%d,%d,%d,%f", &h.Gpio[0], &h.Gpio[1], &h.Gpio[2], &h.Gpio[3], &h.Speed)
	if err != nil {
		return nil, fmt.Errorf("stepper: %v", err)
	}
	if n != 5 {
		return nil, fmt.Errorf("invalid stepper arguments")
	}
	n, err = conf.Parse("steps", "%d", &h.Steps)
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
	n, err = conf.Parse("initial", "%d", &h.Initial)
	if err != nil {
		return nil, fmt.Errorf("initial: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("initial: argument count")
	}
	return &h, nil
}

// setup creates a hand and an encoder.
func setup(hc *ClockConfig) (*Clock, error) {
	var gp [4]*io.Gpio
	var err error
	for i, v := range hc.Gpio {
		gp[i], err = io.OutputPin(v)
		if err != nil {
			return nil, fmt.Errorf("Pin %d: %v", v, err)
		}
	}
	stepper := io.NewStepper(hc.Steps, gp[0], gp[1], gp[2], gp[3])
	mover := &StepperMover{hc.Name, stepper, hc.Speed}
	h := hand.NewHand(hc.Name, hc.Period, mover, hc.Update, int(hc.Steps))
	inp, err := io.Pin(hc.Encoder)
	if err != nil {
		return nil, fmt.Errorf("Encoder %d: %v", hc.Encoder, err)
	}
	enc := hand.NewEncoder(stepper, h, inp, int(hc.Steps))
	return &Clock{h, enc, hc}, nil
}

func (c *Clock) run() {
	hand.Calibrate(c.enc, c.h, c.conf.Steps, 100)
	c.h.Run(c.conf.Initial)
}

// Move moves the stepper motor the steps indicated.
func (m *StepperMover) Move(steps int) {
	if m.stepper != nil {
		m.stepper.Step(m.speed, steps)
		m.stepper.Off() // Turn stepper off between moves
	}
}
