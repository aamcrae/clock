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

// Calibrate and run clock

package hand

import (
	"log"
)

func Calibrate(e *Encoder, h *Hand, initial int) {
	// Calibrate by running at least 2 revolutions to calibrate the encoder.
	h.mover.Move(int(e.reference*2 + e.reference/2))
	if e.Measured == 0 {
		log.Fatalf("Unable to calibrate")
	}
	// Move to encoder reference position.
	loc := e.getStep.GetStep()
	h.mover.Move(int((loc % int64(e.Measured)) + e.Midpoint))
	h.Run(initial)
}
