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
	flag.Int("p1", 4, "GPIO pin for motor output 1"),
	flag.Int("p2", 17, "GPIO pin for motor output 2"),
	flag.Int("p3", 27, "GPIO pin for motor output 3"),
	flag.Int("p4", 22, "GPIO pin for motor output 4"),
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
		defer pins[i].Close()
	}
	stepper := io.NewStepper(halfStepsRev, pins[0], pins[1], pins[2], pins[3])
	defer stepper.Close()
	now := time.Now()
	stepper.Step(*rpm, *steps)
	stepper.Wait()
	log.Printf("Elapsed = %s\n", time.Now().Sub(now))
}
