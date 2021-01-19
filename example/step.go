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

const stepsRev = 2048

var g1 = flag.Int("g1", 4, "GPIO pin for motor input 1")
var g2 = flag.Int("g2", 17, "GPIO pin for motor input 2")
var g3 = flag.Int("g3", 27, "GPIO pin for motor input 3")
var g4 = flag.Int("g4", 22, "GPIO pin for motor input 4")
var rpm = flag.Float64("rpm", 5.0, "RPM")
var steps = flag.Int("steps", stepsRev/12, "Steps")

func main() {
	flag.Parse()
	stepper, err := io.NewStepper(stepsRev, *g1, *g2, *g3, *g4)
	if err != nil {
		log.Fatalf("stepper: %v", err)
	}
	defer stepper.Close()
	now := time.Now()
	st := *steps
	for i := 0; i < 10; i++ {
		stepper.Step(*rpm, st)
		st = -st
	}
	log.Printf("Waiting for completion")
	stepper.Wait()
	log.Printf("Elapsed = %s\n", time.Now().Sub(now))
}
