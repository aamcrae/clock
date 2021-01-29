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
	pwmBaseDir       = "/sys/class/pwm/pwmchip0/"
	pwmExportFile    = pwmBaseDir + "export"
	pwmUnexportFile  = pwmBaseDir + "unexport"
	periodFile     = "/period"
	dutyFile     = "/duty_cycle"
	enableFile     = "/enable"
)


type HwPwm struct {
	unit    int
	base string
	pFile   *os.File
	dFile   *os.File
	period int64
	duty   int64
}

// NewHwPWM creates a new hardware PWM controller.
func NewHwPWM(unit int) (* HwPwm, error) {
	p := new(HwPwm)
	p.unit = unit
	p.base = fmt.Sprintf("%spwm%d", pwmBaseDir, unit)
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
	p.dFile, err = os.OpenFile(fmt.Sprintf("%s%s", p.base, dutyFile), os.O_RDWR, 0600)
	if err != nil {
		p.pFile.Close()
		unexport(pwmUnexportFile, unit)
		return nil, err
	}
	// Default settings
	p.Set(time.Millisecond * 100, 0)
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
	p.pFile.Close()
	p.dFile.Close()
	unexport(pwmUnexportFile, p.unit)
}

// Set sets the PWM parameters.
// Assumes hardware clock is 1 MHz.
func (p *HwPwm) Set(period time.Duration, duty int) error {
	if duty < 0 || duty > 100 {
		return fmt.Errorf("%d: invalid duty cycle percentage")
	}
	pMicro := period.Nanoseconds()
	if pMicro <= 0 {
		return fmt.Errorf("invalid period")
	}
	dMicro := pMicro * int64(duty) / 100
	// When writing the period and duty cycle, the order may be important
	// since duty cycle must not be greater than the current period.
	if dMicro > p.period {
		// Write period first
		_, err := p.pFile.WriteAt([]byte(fmt.Sprintf("%d", pMicro)), 0)
		if err != nil {
			return err
		}
		_, err = p.dFile.WriteAt([]byte(fmt.Sprintf("%d", dMicro)), 0)
		if err != nil {
			return err
		}
	} else {
		if dMicro != p.duty {
			_, err := p.dFile.WriteAt([]byte(fmt.Sprintf("%d", dMicro)), 0)
			if err != nil {
				return err
			}
		}
		if pMicro != p.period {
			_, err := p.pFile.WriteAt([]byte(fmt.Sprintf("%d", pMicro)), 0)
			if err != nil {
				return err
			}
		}
	}
	p.period = pMicro
	p.duty = dMicro
	return nil
}
