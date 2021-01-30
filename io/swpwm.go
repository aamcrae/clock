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

package io

import (
	"fmt"
	"time"
)

type PWM interface {
	Close()
	Set(time.Duration, int) error
}

type pwmMsg struct {
	period time.Duration
	duty   int // Duty cycle as percentage
	stop   chan bool
}

type swPwm struct {
	pin Setter
	c   chan pwmMsg
}

// NewPWM creates a new s/w PWM controller.
func NewSwPWM(pin Setter) *swPwm {
	p := new(swPwm)
	p.pin = pin
	p.c = make(chan pwmMsg, 1)
	go p.handler()
	return p
}

// Close closes the PWM controller
func (p *swPwm) Close() {
	sc := make(chan bool)
	p.c <- pwmMsg{0, 0, sc}
	<-sc
	close(sc)
	close(p.c)
}

// Set sets the PWM parameters. The changes take
// place at the end of the current period.
func (p *swPwm) Set(period time.Duration, duty int) error {
	if duty < 0 || duty > 100 {
		return fmt.Errorf("%d: invalid duty cycle percentage")
	}
	p.c <- pwmMsg{period, duty, nil}
	return nil
}

// goroutine handler
// Listens on message channel, and runs the PWM.
func (p *swPwm) handler() {
	var on, off time.Duration
	off = time.Millisecond * 5
	current := 0
	p.pin.Set(0)
	for {
		if on != 0 {
			if current != 1 {
				p.pin.Set(1)
				current = 1
			}
			time.Sleep(on)
		}
		if off != 0 {
			if current != 0 {
				p.pin.Set(0)
				current = 0
			}
			time.Sleep(off)
		}
		// Check for new parameters after each cycle.
		select {
		case m := <-p.c:
			if m.stop != nil {
				m.stop <- true
				return
			}
			on = m.period * time.Duration(m.duty) / 100
			off = m.period - on
		default:
		}
	}
}
