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

// Program to demonstrate how to watch edge triggered inputs

package main

import (
	"flag"
	"log"

	"github.com/aamcrae/clock/io"
)

var gpio = flag.Int("gpio", 4, "GPIO pin for motor output 1")

func main() {
	flag.Parse()
	p, err := io.Pin(*gpio)
	if err != nil {
		log.Fatalf("Pin %d: %v", *gpio, err)
	}
	err = p.Edge(io.BOTH)
	if err != nil {
		log.Fatalf("Pin %d: edge BOTH: %v", *gpio, err)
	}
	defer p.Close()
	for {
		v, err := p.Get()
		if err != nil {
			log.Fatalf("Pin %d: Get: %v", *gpio, err)
		}
		log.Printf("pin %d = %d\n", *gpio, v)
	}
}
