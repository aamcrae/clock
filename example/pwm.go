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

// Program to demonstrate how to access the s/w PWM library

package main

import (
	"flag"
	"log"
	"math"
	"time"

	"github.com/aamcrae/clock/io"
)

var pwmUnit = flag.Int("pwm", 0, "PWM unit for PWM example")

func main() {
	flag.Parse()
	pwm, err := io.NewHwPWM(*pwmUnit)
	if err != nil {
		log.Fatalf("PWM unit %d: %v", *pwmUnit, err)
	}
	defer pwm.Close()
	for i := 0; i < 10; i++ {
		for v := 0; v < 90; v++ {
			set(pwm, v)
		}
		for v := 89; v >= 0; v-- {
			set(pwm, v)
		}
	}
}

func set(pwm io.PWM, v int) {
	period := time.Millisecond * 1
	r := float64(v) * math.Pi / 180
	d := int(math.Sin(r) * 100.0)
	err := pwm.Set(period, d)
	if err != nil {
		log.Fatalf("Set: period %s, duty %d: %v", period.String(), d, err)
	}
	time.Sleep(time.Millisecond * 50)
}
