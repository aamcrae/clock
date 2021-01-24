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

// HTTP server for clock images
package hand

import (
	"flag"
	"fmt"
    "image"
    "image/jpeg"
	"log"
    "math"
    "os"
	"net/http"

    "github.com/fogleman/gg"
)

var port = flag.Int("port", 8080, "Web server port number")
var refresh = flag.Int("refresh", 10, "Refresh rate in seconds")
var clockface = flag.String("clockface", "clock-face.jpg", "Clock face JPEG file")
const midX = 641
const midY = 646

func ClockServer(h, m, s *Hand) {
    inf, err := os.Open(*clockface)
    if err != nil {
        log.Fatalf("%s: %v", *clockface, err)
    }
    defer inf.Close()
    img, _, err := image.Decode(inf)
    if err != nil {
        log.Fatalf("%s: %v", *clockface, err)
    }
	http.Handle("/clock.jpg", http.HandlerFunc(handler(h, m, s, img)))
	url := fmt.Sprintf(":%d", *port)
	log.Printf("Starting server on %s", url)
	server := &http.Server{Addr: url}
	log.Fatal(server.ListenAndServe())
}

func handler(h, m, s *Hand, img image.Image) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "image/jpeg")
        c := gg.NewContextForImage(img)
        c.SetRGB(0, 0, 1)
        drawHand(c, h, 300, 30)
        drawHand(c, m, 500, 10)
        c.SetRGB(1, 0, 1)
        drawHand(c, s, 600, 2)
        err := jpeg.Encode(w, c.Image(), nil)
        if err != nil {
            log.Printf("Error writing image: %v\n", err)
            w.WriteHeader(http.StatusInternalServerError)
        } else {
            log.Printf("Write jpeg image")
        }

	}
}

func drawHand(c *gg.Context, h *Hand, length, width int) {
    p, r := h.Position()
    p = r - p
    radians := float64(p) * 2 * math.Pi / float64(r) + math.Pi
    x := float64(length) * math.Sin(radians) + float64(midX)
    y := float64(length) * math.Cos(radians) + float64(midY)
    c.SetLineWidth(float64(width))
    c.DrawLine(float64(midX), float64(midY), x, y)
    c.Stroke()
}
