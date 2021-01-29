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

// Package io manages GPIO pins

package io

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// Mode
const (
	IN  = iota // Default
	OUT = iota
)

// Edge
const (
	NONE    = iota // Default
	RISING  = iota
	FALLING = iota
	BOTH    = iota
)

const (
	gpioBaseDir       = "/sys/class/gpio/"
	gpioExportFile    = gpioBaseDir + "export"
	gpioUnexportFile  = gpioBaseDir + "unexport"
	gpioDirectionFile = "/direction"
	gpioValueFile     = "/value"
)

// Gpio represents one GPIO pin.
type Gpio struct {
	number    int
	value     *os.File
	buf       []byte
	direction int
	edge      int
	pollfd    []unix.PollFd
}

// OutputPin opens a GPIO pin and sets the direction as OUTPUT.
func OutputPin(gpio int) (*Gpio, error) {
	g, err := Pin(gpio)
	if err != nil {
		return nil, err
	}
	err = g.Direction(OUT)
	if err != nil {
		g.Close()
		return nil, err
	}
	return g, nil
}

// Pin opens a GPIO pin as an input (by default)
func Pin(gpio int) (*Gpio, error) {
	g := new(Gpio)
	g.number = gpio
	g.buf = make([]byte, 1)

	vFile := fmt.Sprintf("%sgpio%d%s", gpioBaseDir, gpio, gpioValueFile)
	err := export(vFile, gpioExportFile, gpio)
	if err != nil {
		return nil, err
	}
	err = g.Direction(IN)
	if err != nil {
		unexport(gpioUnexportFile, gpio)
		return nil, err
	}
	err = g.Edge(NONE)
	if err != nil {
		unexport(gpioUnexportFile, gpio)
		return nil, err
	}
	g.value, err = os.OpenFile(fmt.Sprintf("%s/gpio%d%s", gpioBaseDir, gpio, gpioValueFile), os.O_RDWR, 0600)
	if err != nil {
		unexport(gpioUnexportFile, gpio)
		return nil, err
	}
	g.pollfd = []unix.PollFd{{int32(g.value.Fd()), unix.POLLPRI | unix.POLLERR, 0}}
	return g, nil
}

// Direction sets the mode (direction) of the GPIO pin.
func (g *Gpio) Direction(d int) error {
	var s string
	switch d {
	case IN:
		s = "in"
	case OUT:
		s = "out"
	default:
		return fmt.Errorf("gpio%d: unknown direction", g.number)
	}
	err := writeFile(fmt.Sprintf("%s/gpio%d/direction", gpioBaseDir, g.number), s)
	if err == nil {
		g.direction = d
	}
	return err
}

// Edge sets the edge detection on the GPIO pin.
func (g *Gpio) Edge(e int) error {
	if g.direction != IN {
		return fmt.Errorf("gpio%d: not set as an input pin", g.number)
	}
	var s string
	switch e {
	case NONE:
		s = "none"
	case RISING:
		s = "rising"
	case FALLING:
		s = "falling"
	case BOTH:
		s = "both"
	default:
		return fmt.Errorf("gpio%d: unknown direction", g.number)
	}
	err := writeFile(fmt.Sprintf("%s/gpio%d/edge", gpioBaseDir, g.number), s)
	if err == nil {
		g.edge = e
	}
	return err
}

// Set the output of the GPIO pin (only valid for OUTPUT pins)
func (g *Gpio) Set(v int) error {
	if g.direction != OUT {
		return fmt.Errorf("gpio%d: is not output", g.number)
	}
	if v == 0 {
		g.buf[0] = '0'
	} else if v == 1 {
		g.buf[0] = '1'
	} else {
		return fmt.Errorf("gpio%d: illegal value", g.number)
	}
	_, err := g.value.WriteAt(g.buf, 0)
	return err
}

// Get returns the current value of the GPIO pin.
func (g *Gpio) Get() (int, error) {
	if g.edge != NONE {
		// Wait for edge using poll.
		for {
			g.pollfd[0].Revents = 0
			_, err := unix.Poll(g.pollfd, -1)
			switch err {
			case nil:
				// Successful call
			case unix.EAGAIN:
				continue
			case unix.EINTR:
				continue
			default:
				return 0, err
			}
			break
		}
		// With no timeout, poll should always return an event.
	}
	_, err := g.value.ReadAt(g.buf, 0)
	if err != nil {
		return 0, err
	}
	if g.buf[0] == '0' {
		return 0, nil
	} else if g.buf[0] == '1' {
		return 1, nil
	} else {
		return 0, fmt.Errorf("gpio%d: unknown value %s", g.number, g.buf)
	}
}

// Close the GPIO pin and unexport it.
func (g *Gpio) Close() {
	g.value.Close()
	unexport(gpioUnexportFile, g.number)
}
