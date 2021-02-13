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
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

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
	defer clk.Close()
	hand.Calibrate(false, clk.Encoder, clk.Hand, clk.Config.Steps)
	reader := bufio.NewReader(os.Stdin)
	enc := clk.Encoder
	var steps int
	measured := enc.Measured
	current := enc.Location()
	steps = diff(measured-hc.Offset, current, measured)
	fmt.Printf("Moving to midnight position (%d steps, %d current, %d offset)\n", steps, current, hc.Offset)
	clk.Move(steps)
	current = (current + steps) % measured
	for {
		fmt.Printf("Location %d (size %d) - offset is %d\n", current, measured, measured-current)
		fmt.Print("Enter steps or command ('help' for help) ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		switch text {
		case "help":
			fmt.Println("  help - print help")
			fmt.Println("  [-]NNN move steps")
			fmt.Println("  q - quit")
		case "q":
			return
		case "o":
			fmt.Printf("Move to original midnight (%d) from %d\n", hc.Offset, current)
			steps = hc.Offset - current
			clk.Move(steps)
			current = (current + steps) % measured
		default:
			n, err := fmt.Sscanf(text, "%d", &steps)
			if err != nil || n != 1 {
				fmt.Printf("Unrecognised input\n")
			} else {
				fmt.Printf("Moving %d steps\n", steps)
				clk.Move(steps)
				current = (current + steps) % measured
			}
		}
	}
}

func diff(a, b, o int) int {
	a %= o
	b %= o
	d := a - b
	if d < 0 {
		d += o
	}
	return d
}
