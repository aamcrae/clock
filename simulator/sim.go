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
	"flag"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/aamcrae/clock/hand"
)

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

var params = []struct {
	name           string
	period, update time.Duration
	reference      int
	perstep        float64
	edge1          int
	edge2          int
}{
	{"hours", 12 * time.Hour, 5 * time.Minute, 4096, 1.003884, 2000, 2199},
	{"minutes", time.Hour, 2 * time.Second, 5123, 1.01234, 3000, 3399},
	{"seconds", time.Minute, 100 * time.Millisecond, 4017, 0.995654, 1500, 1599},
}

const threshold = time.Millisecond * 50

func main() {
	flag.Parse()
	h := sim(0)
	m := sim(1)
	s := sim(2)
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
	go hand.ClockServer(h.hand, m.hand, s.hand)
	for {
		var b strings.Builder
		hval := h.Pos(&b, 1, 12)
		fmt.Fprintf(&b, ":")
		mval := m.Pos(&b, 0, 60)
		fmt.Fprintf(&b, ":")
		sval := s.Pos(&b, 0, 60)
		now := time.Now()
		rt := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()%12, now.Minute(), now.Second(), 0, time.Local)
		myt := time.Date(now.Year(), now.Month(), now.Day(), hval, mval, sval, 0, time.Local)
		diff := myt.Sub(rt)
		if diff > threshold || diff < -threshold {
			fmt.Printf("%s - diff is %s\n", b.String(), diff.String())
		}
		time.Sleep(time.Second * 5)
	}
}

func (s *SimHand) Pos(w io.Writer, offs, units int) int {
	p, r := s.hand.Position()
	v := p * units / r
	fmt.Fprintf(w, "%02d", v)
	return v
}

func sim(index int) *SimHand {
	p := &params[index]
	sh := new(SimHand)
	sh.encChan = make(chan int, 10)
	sh.reference = int64(p.reference)
	sh.perstep = p.perstep
	sh.actual = float64(p.reference) * p.perstep
	sh.edge1 = p.edge1
	sh.edge2 = p.edge2
	sh.hand = hand.NewHand(p.name, p.period, sh, p.update, p.reference)
	sh.encoder = hand.NewEncoder(sh, sh.hand, sh, p.edge1-p.edge2+1)
	go hand.Calibrate(sh.encoder, sh.hand, p.reference, (p.edge1+p.edge2+1)/2)
	return sh
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
