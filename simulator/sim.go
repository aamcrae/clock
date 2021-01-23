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
	"log"
	"time"

	"github.com/aamcrae/clock"
	"github.com/aamcrae/config"
)

var startTime = flag.String("time", "3:04:05", "Current time on clock face")
var configFile = flag.String("config", "", "Configuration file")

type SimHand struct {
}

type Sim struct {
}

func main() {
	flag.Parse()
}
