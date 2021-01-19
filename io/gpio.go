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

const (
	baseDir       = "/sys/class/gpio/"
	exportFile    = baseDir + "export"
	unexportFile  = baseDir + "unexport"
	directionFile = "/direction"
	valueFile     = "/value"
)

const verifyTimeout = 2 * time.Second

// Verify will wait for exported files to become writable.
// This is necessary if the process is not running as root, since systemd
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
	g.value, err = os.OpenFile(fmt.Sprintf("%s/gpio%d%s", baseDir, gpio, valueFile), os.O_RDWR, 0600)
	if err != nil {
		unexport(gpio)
		return nil, err
	}
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
