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

// Program to demonstrate how to access the stepper motor library.

package main

import (
	"flag"
	"log"
	"time"

	"github.com/aamcrae/clock/io"
)

const halfStepsRev = 2048 * 2

var gpios = []*int{
	flag.Int("a1", 4, "GPIO pin for motor A input 1"),
	flag.Int("a2", 17, "GPIO pin for motor A input 2"),
	flag.Int("a3", 27, "GPIO pin for motor A input 3"),
	flag.Int("a4", 22, "GPIO pin for motor A input 4"),
	flag.Int("b1", 6, "GPIO pin for motor B input 1"),
	flag.Int("b2", 13, "GPIO pin for motor B input 2"),
	flag.Int("b3", 19, "GPIO pin for motor B input 3"),
	flag.Int("b4", 26, "GPIO pin for motor B input 4"),
}
var rpm = flag.Float64("rpm", 5.0, "RPM")
var steps = flag.Int("steps", halfStepsRev/12, "Steps")

func main() {
	flag.Parse()
	pins := make([]*io.Gpio, len(gpios))
	for i, gp := range gpios {
		var err error
		pins[i], err = io.OutputPin(*gp)
		if err != nil {
			log.Fatalf("Pin %d: %v", *gp, err)
		}
	}
	stepperA := io.NewStepper(halfStepsRev, pins[0], pins[1], pins[2], pins[3])
	stepperB := io.NewStepper(halfStepsRev, pins[4], pins[5], pins[6], pins[7])
	defer stepperA.Close()
	defer stepperB.Close()
	stepperA.Restore(0)
	stepperB.Restore(0)
	now := time.Now()
	st := *steps
	for i := 0; i < 10; i++ {
		stepperA.Step(*rpm, st)
		stepperB.Step(*rpm, -st)
		st = -st
	}
	time.Sleep(4 * time.Second)
	log.Printf("Stopping A")
	stepperA.Stop()
	log.Printf("Waiting for completion")
	stepperA.Wait()
	stepperB.Wait()
	log.Printf("Elapsed = %s, index A = %d, index B = %d\n", time.Now().Sub(now), stepperA.Save(), stepperB.Save())
}
