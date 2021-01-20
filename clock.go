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

type StepperMover struct {
	name    string
	stepper *io.Stepper
	speed   float64
	total   int
}

type FakeMover struct {
	name  string
	total int
}

var startTime = flag.String("time", "3:04:05", "Current time on clock face")
var speed = flag.Float64("speed", 4.0, "Stepper speed in RPM")
var halfSteps = flag.Float64("steps", 2048*2, "Half steps in a revolution")

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
	mh := &StepperMover{"hours", io.NewStepper(*halfSteps, pins[4], pins[5], pins[6], pins[7]), *speed, 0}
	hour := NewHand("hours", time.Hour*12, mh, time.Minute*5, int(*halfSteps))

	mm := &StepperMover{"minutes", io.NewStepper(*halfSteps, pins[0], pins[1], pins[2], pins[3]), *speed, 0}
	min := NewHand("minutes", time.Hour, mm, time.Second*10, int(*halfSteps))

	ms := &FakeMover{"Seconds", 0}
	sec := NewHand("seconds", time.Minute, ms, time.Millisecond*250, int(*halfSteps))

	hour.Start(face)
	min.Start(face)
	sec.Start(face)
	select {}
}

func (m *StepperMover) Move(steps int) {
	//m.stepper.Step(steps)
	//m.stepper.Off(steps)		// Turn stepper off between moves
	m.total += steps
	fmt.Printf("%s step %d, total %d\n", m.name, steps, m.total)
}

func (m *FakeMover) Move(steps int) {
	m.total += steps
	fmt.Printf("%s: step %d, total %d\n", m.name, steps, m.total)
}
