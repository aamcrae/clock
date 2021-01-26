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

package hand

import (
	"fmt"
	"log"
	"time"

	"github.com/aamcrae/clock/io"
	"github.com/aamcrae/config"
)

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

type ClockHand struct {
	stepper *io.Stepper
	Hand    *Hand
	Encoder *Encoder
	Config  *ClockConfig
}

// Config reads and validates a hand config from a config file section.
// Sample config:
//  stepper=4,17,27,22,3.0
//  period=12h
//  update=5m
//  steps=4096
//  initial=2100
func Config(conf *config.Config, name string) (*ClockConfig, error) {
	s := conf.GetSection(name)
	if s == nil {
		return nil, fmt.Errorf("no config for %s", name)
	}
	var err error
	var h ClockConfig
	h.Name = name
	h.Gpio = make([]int, 4)
	n, err := s.Parse("stepper", "%d,%d,%d,%d,%f", &h.Gpio[0], &h.Gpio[1], &h.Gpio[2], &h.Gpio[3], &h.Speed)
	if err != nil {
		return nil, fmt.Errorf("stepper: %v", err)
	}
	if n != 5 {
		return nil, fmt.Errorf("invalid stepper arguments")
	}
	n, err = s.Parse("steps", "%d", &h.Steps)
	if err != nil {
		return nil, fmt.Errorf("steps: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("steps: argument count")
	}
	p, err := s.GetArg("period")
	if err != nil {
		return nil, fmt.Errorf("period: %v", err)
	}
	h.Period, err = time.ParseDuration(p)
	if err != nil {
		return nil, fmt.Errorf("period: %v", err)
	}
	u, err := s.GetArg("update")
	if err != nil {
		return nil, fmt.Errorf("update: %v", err)
	}
	h.Update, err = time.ParseDuration(u)
	if err != nil {
		return nil, fmt.Errorf("update: %v", err)
	}
	n, err = s.Parse("encoder", "%d", &h.Encoder)
	if err != nil {
		return nil, fmt.Errorf("encoder: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("encoder: argument count")
	}
	n, err = s.Parse("initial", "%d", &h.Initial)
	if err != nil {
		return nil, fmt.Errorf("initial: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("initial: argument count")
	}
	return &h, nil
}

// NewClockHand initialises the stepper and encoder from the
// clock configuration.
func NewClockHand(hc *ClockConfig) (*ClockHand, error) {
	c := new(ClockHand)
	c.Config = hc
	var gp [4]*io.Gpio
	var err error
	for i, v := range hc.Gpio {
		gp[i], err = io.OutputPin(v)
		if err != nil {
			return nil, fmt.Errorf("Pin %d: %v", v, err)
		}
	}
	c.stepper = io.NewStepper(hc.Steps, gp[0], gp[1], gp[2], gp[3])
	h := NewHand(hc.Name, hc.Period, c, hc.Update, int(hc.Steps))
	inp, err := io.Pin(hc.Encoder)
	if err != nil {
		c.stepper.Close()
		return nil, fmt.Errorf("Encoder %d: %v", hc.Encoder, err)
	}
	err = inp.Edge(io.BOTH)
	if err != nil {
		c.stepper.Close()
		inp.Close()
		return nil, fmt.Errorf("Encoder %d: %v", hc.Encoder, err)
	}
	c.Encoder = NewEncoder(c.stepper, h, inp, int(hc.Steps))
	return c, nil
}

func (c *ClockHand) Run() {
	Calibrate(true, c.Encoder, c.Hand, c.Config.Steps, c.Config.Initial)
}

// Move moves the stepper motor the steps indicated.
func (c *ClockHand) Move(steps int) {
	if c.stepper != nil {
		c.stepper.Step(c.Config.Speed, steps)
		c.stepper.Off() // Turn stepper off between moves
	}
}

// Calibrate moves the hand at least 2 1/2 revolutions to
// allow the encoder to measure the actual steps required
// for 360 degrees of movement.
// Once that is known, the hand is moved to the midpoint of the encoder,
// and this is considered the reference point for the hand.
func Calibrate(run bool, e *Encoder, h *Hand, reference, initial int) {
	log.Printf("%s: Starting calibration", h.Name)
	h.mover.Move(int(reference*2 + reference/2))
	if e.Measured == 0 {
		log.Fatalf("Unable to calibrate")
	}
	// Move to encoder reference position.
	loc := e.getStep.GetStep()
	log.Printf("%s: Calibration complete, moving to encoder midpoint (%d)", h.Name, e.Midpoint%e.Measured)
	h.mover.Move(int((loc % int64(e.Measured))) + e.Midpoint)
	// The hand is at the midpoint of the encoder, so the hand is
	// at a known physical location, which is set as the initial position.
	h.Current = initial
	if run {
		h.Run()
	}
}
