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

// clock simulator program

package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aamcrae/clock/hand"
)

// Simulator framework for 1 hand.
// The input to the encoder is simulated via a channel that
// has values sent to it when the simulated stepper steps past the
// edges of an encoder mark.
type SimHand struct {
	hand         *hand.Hand
	encoder      *hand.Encoder
	current      float64
	steps        int64
	encChan      chan int
	encValue     int
	edge1, edge2 int
	reference    int
	perstep      float64
	actual       float64
}

// Hand parameters
var params = []struct {
	name           string        // Name of hand
	period, update time.Duration // Period of 1 revolution
	reference      int           // Reference steps per revolution
	perstep        float64       // factor for actual steps
	edge1          int           // Position of encoder mark edge 0->1
	edge2          int           // Position of encoder mark edge 1->0
	offset         int           // Offset of hand from encoder
	units          int           // Units of hand
}{
	{"hours", 12 * time.Hour, 1 * time.Minute, 4096, 1.003884, 2000, 2199, 1000, 12},
	{"minutes", time.Hour, 2 * time.Second, 5123, 1.01234, 3000, 3399, 2000, 60},
	{"seconds", time.Minute, 100 * time.Millisecond, 4017, 0.995654, 1500, 1599, 3000, 60},
}

const threshold = time.Millisecond * 50

var port = flag.Int("port", 8080, "Web server port number")

func main() {
	flag.Parse()
	var hands []*SimHand
	for i := range params {
		hands = append(hands, sim(i))
	}
	for {
		ready := 0
		for _, s := range hands {
			if s.hand.Ticking {
				ready++
			}
		}
		if ready == len(hands) {
			break
		}
		time.Sleep(time.Second)
		fmt.Printf("Waiting for initialisation to complete (%d/%d ready)\n", ready, len(hands))
	}
	fmt.Printf("Clock initialisation complete\n")
	var clk []*hand.Hand
	for _, sh := range hands {
		clk = append(clk, sh.hand)
	}
	go hand.ClockServer(*port, clk)
	for {
		var b strings.Builder
		var val [3]int
		for i, h := range hands {
			val[i] = h.Pos(&b, params[i].units)
			fmt.Fprintf(&b, ":")
		}
		now := time.Now()
		rt := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()%12, now.Minute(), now.Second(), 0, time.Local)
		myt := time.Date(now.Year(), now.Month(), now.Day(), val[0], val[1], val[2], 0, time.Local)
		diff := myt.Sub(rt)
		if diff > threshold || diff < -threshold {
			fmt.Printf("%s - diff is %s\n", b.String(), diff.String())
		}
		time.Sleep(time.Second * 5)
	}
}

// Pos writes the current value of the hand as determined from
// the hand position on the dial.
func (s *SimHand) Pos(w io.Writer, units int) int {
	p, r, _ := s.hand.Get()
	v := p * units / r
	fmt.Fprintf(w, "%02d", v)
	return v
}

// Create and initialise a hand simulator, and
// start it running.
func sim(index int) *SimHand {
	p := &params[index]
	sh := new(SimHand)
	sh.encChan = make(chan int, 10)
	sh.encChan <- 0
	sh.reference = p.reference
	sh.perstep = p.perstep
	sh.edge1 = p.edge1
	sh.edge2 = p.edge2
	sh.hand = hand.NewHand(p.name, p.period, sh, p.update, p.reference, p.offset)
	sh.encoder = hand.NewEncoder(p.name, sh, sh.hand, sh, (p.edge2-p.edge1+1)/2)
	go hand.Calibrate(true, sh.encoder, sh.hand, p.reference)
	return sh
}

// Move acts like a stepper motor, moving the hand
// one step at a time, and checking whether the encoder mark
// is triggered. The actual size of each step is not integral to
// simulate a hand revolution that's not exactly an integral size.
// The idea is that the encoder will correct the revolution size
// so that errors do not build up.
func (s *SimHand) Move(steps int) {
	var e1, e2 int
	var sInc int64
	var inc float64
	if steps < 0 {
		// CCW
		inc = -s.perstep
		sInc = -1
		e1 = s.edge1 - 1
		e2 = s.edge2
		steps = -steps
	} else {
		inc = s.perstep
		sInc = 1
		e1 = s.edge1
		e2 = s.edge2 + 1
	}
	for i := 0; i < steps; i++ {
		s.current += inc
		s.steps += sInc
		loc := int(s.current) % s.reference
		// Check for stepping hitting an encoder edge.
		if loc == e1 || loc == e2 {
			s.encValue ^= 1
			s.encChan <- s.encValue
		}
		time.Sleep(time.Microsecond * 20)
		//time.Sleep(time.Millisecond)
	}
}

// GetLocation returns the current step location
func (s *SimHand) GetLocation() int64 {
	return s.GetStep()
}

// GetStep returns the current step location
func (s *SimHand) GetStep() int64 {
	return s.steps
}

// Get returns an encoder I/O value when
// it changes.
func (s *SimHand) Get() (int, error) {
	for {
		v := <-s.encChan
		return v, nil
	}
}
