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
	"os"
	"time"
)

const (
	pwmBaseDir      = "/sys/class/pwm/pwmchip0/"
	pwmExportFile   = pwmBaseDir + "export"
	pwmUnexportFile = pwmBaseDir + "unexport"
	periodFile      = "/period"
	dutyFile        = "/duty_cycle"
	enableFile      = "/enable"
)

type HwPwm struct {
	unit   int
	base   string
	pFile  *os.File
	dFile  *os.File
	period int64
	duty   int64
}

// NewHwPWM creates a new hardware PWM controller.
func NewHwPWM(unit int) (*HwPwm, error) {
	p := new(HwPwm)
	p.unit = unit
	p.base = fmt.Sprintf("%spwm%d", pwmBaseDir, unit)
	p.period = -1
	p.duty = -1

	vFile := fmt.Sprintf("%s%s", p.base, periodFile)
	err := export(vFile, pwmExportFile, unit)
	if err != nil {
		return nil, err
	}
	p.pFile, err = os.OpenFile(fmt.Sprintf("%s%s", p.base, periodFile), os.O_RDWR, 0600)
	if err != nil {
		unexport(pwmUnexportFile, unit)
		return nil, err
	}
	dName := fmt.Sprintf("%s%s", p.base, dutyFile)
	err = verifyFile(dName)
	if err != nil {
		p.pFile.Close()
		unexport(pwmUnexportFile, unit)
		return nil, err
	}
	p.dFile, err = os.OpenFile(dName, os.O_RDWR, 0600)
	if err != nil {
		p.pFile.Close()
		unexport(pwmUnexportFile, unit)
		return nil, err
	}
	// Default settings
	p.Set(time.Millisecond*100, 0)
	err = writeFile(fmt.Sprintf("%s%s", p.base, enableFile), "1")
	if err != nil {
		p.pFile.Close()
		p.dFile.Close()
		unexport(pwmUnexportFile, unit)
		return nil, err
	}
	return p, nil
}

// Close closes the PWM controller
func (p *HwPwm) Close() {
	writeFile(fmt.Sprintf("%s%s", p.base, enableFile), "0")
	p.pFile.Close()
	p.dFile.Close()
	unexport(pwmUnexportFile, p.unit)
}

// Set sets the PWM parameters.
func (p *HwPwm) Set(period time.Duration, duty int) error {
	if duty < 0 || duty > 100 {
		return fmt.Errorf("%d: invalid duty cycle percentage")
	}
	pNano := period.Nanoseconds()
	if pNano < 15 {
		return fmt.Errorf("invalid period")
	}
	dNano := pNano * int64(duty) / 100
	// When writing the period and duty cycle, the order may be important
	// since duty cycle must not be greater than the current period.
	if dNano > p.period {
		// Write period first
		_, err := p.pFile.WriteAt([]byte(fmt.Sprintf("%d", pNano)), 0)
		if err != nil {
			return err
		}
		_, err = p.dFile.WriteAt([]byte(fmt.Sprintf("%d", dNano)), 0)
		if err != nil {
			return err
		}
	} else {
		if dNano != p.duty {
			_, err := p.dFile.WriteAt([]byte(fmt.Sprintf("%d", dNano)), 0)
			if err != nil {
				return err
			}
		}
		if pNano != p.period {
			_, err := p.pFile.WriteAt([]byte(fmt.Sprintf("%d", pNano)), 0)
			if err != nil {
				return err
			}
		}
	}
	p.period = pNano
	p.duty = dNano
	return nil
}
