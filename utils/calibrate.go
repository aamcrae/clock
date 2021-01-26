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

// Calibration utility

package main

import (
	"flag"
	"log"

	"github.com/aamcrae/clock/hand"
	"github.com/aamcrae/config"
)

var configFile = flag.String("config", "", "Configuration file")
var section = flag.String("hand", "", "Hand to calibrate e.g hours, minutes, seconds")

func main() {
	flag.Parse()
	conf, err := config.ParseFile(*configFile)
	if err != nil {
		log.Fatalf("%s: %v", *configFile, err)
	}
	hc, err := hand.Config(conf, *section)
	if err != nil {
		log.Fatalf("%s: %v", *configFile, err)
	}
	clk, err := hand.NewClockHand(hc)
	if err != nil {
		log.Fatalf("ClockHand: %s %v", *section, err)
	}
	hand.Calibrate(false, clk.Encoder, clk.Hand, clk.Config.Steps, clk.Config.Initial)
}
