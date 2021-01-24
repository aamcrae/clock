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
	"math"
	"time"

	"github.com/aamcrae/clock/hand"
)

const refSteps = 4096

type SimHand struct {
	hand         *hand.Hand
	encoder      *hand.Encoder
	current      float64
	encChan      chan int
	encValue     int
	edge1, edge2 int
	reference    int64
	perstep      float64
	actual       float64
}

type Sim struct {
}

func main() {
	h := sim("hours", 12*time.Hour, 5*time.Minute, 4096, 1.003884)
	m := sim("minutes", time.Hour, 10*time.Second, 4096, 1.01234)
	s := sim("seconds", time.Minute, 250*time.Millisecond, 4096, 0.997654)
	for {
		hands := 0
		if h.encoder.Measured != 0 {
			hands++
		}
		if m.encoder.Measured != 0 {
			hands++
		}
		if s.encoder.Measured != 0 {
			hands++
		}
		if hands == 3 {
			break
		}
		time.Sleep(time.Second)
		fmt.Printf("Waiting for calibration\n")
	}
	for {
		hval := h.Pos(1, 12)
		fmt.Printf(":")
		mval := m.Pos(0, 60)
		fmt.Printf(":")
		sval := s.Pos(0, 60)
		now := time.Now()
		rt := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()%12, now.Minute(), now.Second(), 0, time.Local)
		myt := time.Date(now.Year(), now.Month(), now.Day(), hval, mval, sval, 0, time.Local)
		fmt.Printf(" - diff is %s\n", myt.Sub(rt).String())
		time.Sleep(time.Second * 5)
	}
}

func (s *SimHand) Pos(offs, units int) int {
	p, r := s.hand.Position()
	v := p * units / r
	fmt.Printf("%02d", v)
	return v
}

func sim(name string, period, update time.Duration, ref int, perstep float64) *SimHand {
	sh := new(SimHand)
	sh.encChan = make(chan int, 10)
	sh.reference = int64(ref)
	sh.perstep = perstep
	sh.actual = float64(ref) * perstep
	sh.edge1 = 2000
	sh.edge2 = 2200
	sh.hand = hand.NewHand(name, period, sh, update, ref)
	sh.encoder = hand.NewEncoder(sh, sh.hand, sh, ref, 100)
	go hand.Calibrate(sh.encoder, sh.hand, 2100)
	return sh
}

func (s *SimHand) Start() {
	// Start by calibrating the encoder
	s.hand.Run(0)
	fmt.Printf("current = %d, measured = %d\n", int(s.current), s.encoder.Measured)
}

func (s *SimHand) Move(steps int) {
	var e1, e2 int
	var inc float64
	if steps < 0 {
		// CCW
		inc = -s.perstep
		e1 = s.edge1 - 1
		e2 = s.edge2
		steps = -steps
	} else {
		inc = s.perstep
		e1 = s.edge1
		e2 = s.edge2 + 1
	}
	for i := 0; i < steps; i++ {
		s.current += inc
		loc := int(math.Mod(s.current, s.actual))
		if loc == e1 || loc == e2 {
			s.encValue ^= 1
			s.encChan <- s.encValue
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *SimHand) GetStep() int64 {
	return int64(s.current)
}

// Block waiting for input value
func (s *SimHand) Get() (int, error) {
	for {
		v := <-s.encChan
		return v, nil
	}
}
