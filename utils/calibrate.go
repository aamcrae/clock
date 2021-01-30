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
	hand.Calibrate(false, clk.Encoder, clk.Hand, clk.Config.Steps, clk.Config.Initial)
	reader := bufio.NewReader(os.Stdin)
	st := clk.Stepper
	enc := clk.Encoder
	var steps int
	for {
		at := int(st.GetStep() % int64(enc.Measured))
		fmt.Printf("Absolute: %d, relative %d (size %d), midpoint %d (offset %d)\n", st.GetStep(), at, enc.Measured, enc.Midpoint, diff(at, enc.Midpoint, enc.Measured))
		fmt.Print("Enter steps or command ('help' for help) ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		switch text {
		case "help":
			fmt.Println("  help - print help")
			fmt.Println("  [-]NNN move steps")
			fmt.Println("  m - move to encoder midpoint")
			fmt.Println("  q - quit")
		case "q":
			return
		case "m":
			fmt.Printf("Move to midpoint (%d) from %d\n", enc.Midpoint, st.GetStep())
			steps = diff(int(st.GetStep()), enc.Midpoint, enc.Measured)
			clk.Move(steps)
		default:
			n, err := fmt.Sscanf(text, "%d", &steps)
			if err != nil || n != 1 {
				fmt.Printf("Unrecognised input\n")
			} else {
				fmt.Printf("Moving %d steps\n", steps)
				clk.Move(steps)
			}
		}
	}
}

func diff(a, b, o int) int {
	a %= o
	b %= o
	d := b - a
	if d < 0 {
		d += o
	}
	return d
}
