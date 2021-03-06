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

	"github.com/aamcrae/config"
	"github.com/aamcrae/gpio"
	"github.com/aamcrae/gpio/action"
)

// Configuration data for the clock hand, usually read from a configuration file.
type ClockConfig struct {
	Name    string        // Name of the hand
	Gpio    []int         // Output pins for the stepper
	Speed   float64       // Speed the stepper runs at (RPM)
	Period  time.Duration // Period of the hand (e.g time.Hour)
	Update  time.Duration // How often the hand updates.
	Steps   int           // Initial reference steps per revolution
	Encoder int           // Input pin for encoder
	Notch   int           // Minimum width of encoder mark
	Offset  int           // Hand offset from midnight to encoder mark
}

// ClockHand combines the I/O for a hand and an encoder.
// A clock is comprised of multiple hands, each of which runs independently.
// Each clock hand consists of a Hand which generates move requests according to the current time,
// an Encoder which provides feedback as to the actual location of the hand,
// and the I/O providers for the Hand and Encoder.
// A config for each hand is parsed from a configuration file.
type ClockHand struct {
	Stepper *action.Stepper
	Input   *io.Gpio
	Hand    *Hand
	Encoder *Encoder
	Config  *ClockConfig
}

// Config reads and validates a ClockHand config from a config file section.
// Sample config:
//  [name]                   # name of hand e.g hours, minutes, seconds
//  stepper=4,17,27,22,3.0   # GPIOs for stepper motor, and speed in RPM
//  period=12h               # The clock period for this hand
//  update=5m                # The update rate as a duration
//  steps=4096               # Reference number of steps in a revolution
//  encoder=21               # GPIO for encoder
//  notch=100                # Min width of sensor mark
//  offset=2100              # The offset of the hand at the encoder mark
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
	n, err = s.Parse("notch", "%d", &h.Notch)
	if err != nil {
		return nil, fmt.Errorf("notch: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("notch: argument count")
	}
	n, err = s.Parse("offset", "%d", &h.Offset)
	if err != nil {
		return nil, fmt.Errorf("offset: %v", err)
	}
	if n != 1 {
		return nil, fmt.Errorf("offset: argument count")
	}
	return &h, nil
}

// NewClockHand initialises the I/O, Hand, and Encoder using the configuration provided.
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
	c.Stepper = action.NewStepper(hc.Steps, gp[0], gp[1], gp[2], gp[3])
	c.Hand = NewHand(hc.Name, hc.Period, c, hc.Update, int(hc.Steps), hc.Offset)
	c.Input, err = io.Pin(hc.Encoder)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("Encoder %d: %v", hc.Encoder, err)
	}
	err = c.Input.Edge(io.BOTH)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("Encoder %d: %v", hc.Encoder, err)
	}
	c.Encoder = NewEncoder(hc.Name, c.Stepper, c.Hand, c.Input, hc.Notch)
	return c, nil
}

// Run starts the clock hand, initially running a calibration so that
// the encoder mark position can be discovered, and then starting the
// hand processing if requested.
func (c *ClockHand) Run() {
	Calibrate(true, c.Encoder, c.Hand, c.Config.Steps)
}

// Move moves the stepper motor the steps indicated. This is a
// shim between the hand and the stepper so that the motor can be
// turned off between movements. Waits until the motor completes the
// steps before returning.
// TODO: Turning the motor off immediately will miss steps under load, so
// some kind of delay is needed.
func (c *ClockHand) Move(steps int) {
	if c.Stepper != nil {
		c.Stepper.Step(c.Config.Speed, steps)
		c.Stepper.Wait()
	}
}

// GetLocation returns the current absolute location.
func (c *ClockHand) GetLocation() int64 {
	return c.Stepper.GetStep()
}

// Close shuts down the clock hand and release the resources.
func (c *ClockHand) Close() {
	if c.Stepper != nil {
		c.Stepper.Close()
	}
	if c.Input != nil {
		c.Input.Close()
	}
}

// Calibrate moves the hand at least 4 revolutions to allow
// the encoder to measure the actual steps for 360 degrees of movement, and
// to discover the location of the encoder mark.
func Calibrate(run bool, e *Encoder, h *Hand, reference int) {
	log.Printf("%s: Starting calibration", h.Name)
	h.mover.Move(int(reference*4 + reference/2))
	if e.Measured == 0 {
		log.Fatalf("Unable to calibrate")
	}
	log.Printf("%s: Calibration complete (%d steps), encoder: %d", h.Name, e.Measured, e.Location())
	if run {
		h.Run()
	}
}
