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

// Simulator clock program

package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aamcrae/clock"
)

const refSteps = 4096

type SimHand struct {
	current      uint64
	encChan      chan int
	encValue     int
	edge1, edge2 int
	reference    int64
	actual       int64
}

type Sim struct {
}

func main() {
	flag.Parse()
	h := sim("hours", 12*time.Hour, 5*time.Minute, 4096, 5003)
	m := sim("minutes", time.Hour, 10*time.Second, 4096, 4090)
	s := sim("seconds", time.Minute, 250*time.Millisecond, 4096, 4099)
}

func sim(name string, period, update time.Duration, ref, actual int) *SimHand {
	sh := new(SimHand)
	sh.encChan = make(chan int, 10)
	sh.reference = int64(ref)
	sh.actual = int64(actual)
	sh.edge1 = 2000
	sh.edge2 = 2200
	h := NewHand(name, period, sh, update, ref)
	NewEncoder(sh, h, sh, steps, 100)
	h.Start(3000)
}

func (s *SimHand) Move(steps int) {
	var e1, e2 int
	var inc int64
	if steps < 0 {
		// CCW
		inc = -1
		e1 = s.edge1 - 1
		e2 = s.edge2
		steps = -steps
	} else {
		inc = 1
		e1 = s.edge1
		e2 = s.edge2 + 1
	}
	for i := 0; i < steps; i++ {
		s.current += inc
		loc := int(s.current % s.actual)
		if loc == e1 || loc == e2 {
			sh.encValue ^= 1
			encChan <- sh.encValue
		}
	}
}

func (s *SimHand) GetStep() int64 {
	return s.current
}

// Block waiting for input value
func (s *SimHand) Get() (int, error) {
	for {
		return <-s.encChan, nil
	}
}
