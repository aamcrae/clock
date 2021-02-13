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

// Main clock program

package main

import (
	"flag"
	"log"

	"github.com/aamcrae/clock/hand"
	"github.com/aamcrae/config"
)

var configFile = flag.String("config", "", "Configuration file")
var port = flag.Int("port", 8080, "Web server port number")

func main() {
	flag.Parse()
	conf, err := config.ParseFile(*configFile)
	if err != nil {
		log.Fatalf("%s: %v", *configFile, err)
	}
	// Read the configs for each of the hands, and create
	// a ClockHand for each config that is found.
	var clock []*hand.ClockHand
	for _, sect := range []string{"hours", "minutes", "seconds"} {
		hc, err := hand.Config(conf, sect)
		if err != nil {
			log.Printf("Invalid config for %s (%v), skipping", sect, err)
			continue
		}
		c, err := hand.NewClockHand(hc)
		if err != nil {
			log.Fatalf("%s: %v", hc.Name, err)
		}
		clock = append(clock, c)
	}
	// Start the clock hands.
	for _, c := range clock {
		go c.Run()
	}
	if len(clock) == 0 {
		log.Fatalf("No clock hands to run!")
	}
	// Start a status server that can display a clock face reflecting the
	// status of the clock.
	if *port != 0 {
		var hands []*hand.Hand
		for _, c := range clock {
			hands = append(hands, c.Hand)
		}
		hand.ClockServer(*port, hands)
	}
	select {}
}
