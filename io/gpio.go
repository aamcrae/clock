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
	"os/user"
	"time"

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
	baseDir       = "/sys/class/gpio/"
	exportFile    = baseDir + "export"
	unexportFile  = baseDir + "unexport"
	directionFile = "/direction"
	valueFile     = "/value"
)

const verifyTimeout = 2 * time.Second

// Verify will wait for exported files to become writable.
// This is necessary if the process is not running as root - systemd
// and udev will change the group permissions on the exported files, but
// this takes some time to do. If we try and access the files before
// the file group/modes are changed, we will get a permission error.
var Verify = false

// Gpio represents one GPIO pin.
type Gpio struct {
	number    int
	value     *os.File
	buf       []byte
	direction int
	edge      int
	pollfd    []unix.PollFd
}

func init() {
	// If the user is not root, enable Verify mode
	u, err := user.Current()
	if err == nil && u.Uid != "0" {
		Verify = true
	}
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

	err := export(g.number)
	if err != nil {
		return nil, err
	}
	err = g.Direction(IN)
	if err != nil {
		unexport(gpio)
		return nil, err
	}
	err = g.Edge(NONE)
	if err != nil {
		unexport(gpio)
		return nil, err
	}
	g.value, err = os.OpenFile(fmt.Sprintf("%s/gpio%d%s", baseDir, gpio, valueFile), os.O_RDWR, 0600)
	if err != nil {
		unexport(gpio)
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
	err := writeFile(fmt.Sprintf("%s/gpio%d/direction", baseDir, g.number), s)
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
	err := writeFile(fmt.Sprintf("%s/gpio%d/edge", baseDir, g.number), s)
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
		g.pollfd[0].Revents = 0
		_, err := unix.Poll(g.pollfd, -1)
		if err != nil {
			return 0, err
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
	unexport(g.number)
}

func unexport(g int) error {
	return writeFile(unexportFile, fmt.Sprintf("%d", g))
}

func export(g int) error {
	// Check if directory and files already exist.
	val := fmt.Sprintf("%s/gpio%d%s", baseDir, g, valueFile)
	err := unix.Access(val, unix.W_OK|unix.R_OK)
	if err == nil {
		return nil
	}
	err = writeFile(exportFile, fmt.Sprintf("%d", g))
	if err == nil && Verify {
		return verifyFile(val)
	}
	return err
}

func writeFile(fname, s string) error {
	f, err := os.OpenFile(fname, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(s))
	return err
}

// Wait for file to become writable
func verifyFile(f string) error {
	var tout time.Duration
	sl := time.Millisecond
	for tout = 0; tout < verifyTimeout; tout += sl {
		err := unix.Access(f, unix.W_OK)
		if err == nil {
			return nil
		}
		time.Sleep(sl)
	}
	return fmt.Errorf("%s: not writable", f)
}
