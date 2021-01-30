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

// Package io common constants and functions

package io

import (
	"fmt"
	"os"
	"os/user"
	"time"

	"golang.org/x/sys/unix"
)

// Setter is an interface for setting an output value on a GPIO
type Setter interface {
	Set(int) error
}

const verifyTimeout = 2 * time.Second

// Verify will enable waiting for exported files to become writable.
// This is necessary if the process is not running as root - systemd
// and udev will change the group permissions on the exported files, but
// this takes some time to do. If we try and access the files before
// the file group/modes are changed, we will get a permission error.
// This can be overridden.
var Verify = false

func init() {
	// If the user is not root, enable Verify mode
	u, err := user.Current()
	if err == nil && u.Uid != "0" {
		Verify = true
	}
}

// unexport writes a unit number to an unexport file.
func unexport(f string, g int) error {
	return writeFile(f, fmt.Sprintf("%d", g))
}

// export will check for the existence of a file, and if it is
// not writable, will write a unit number to an export file, and then
// optionally wait for the file to appear and become writable.
func export(f, expfile string, g int) error {
	// Check if directory and files already exist.
	err := unix.Access(f, unix.W_OK|unix.R_OK)
	if err == nil {
		return nil
	}
	err = writeFile(expfile, fmt.Sprintf("%d", g))
	if err == nil && Verify {
		return verifyFile(f)
	}
	return err
}

// Write a string to a file.
func writeFile(fname, s string) error {
	f, err := os.OpenFile(fname, os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(s))
	return err
}

// Wait for file to become writable.
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
