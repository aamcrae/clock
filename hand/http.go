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
	"net/http"
	"os"

	"github.com/fogleman/gg"
)

var clockface = flag.String("clockface", "clock-face.jpg", "Clock face JPEG file")
var refresh = flag.Int("refresh", 30, "Refresh status page number of seconds")

var handDraw = []struct {
	r      float64
	g      float64
	b      float64
	length int
	width  int
}{
	{0, 0, 1, 400, 30},
	{0, 0, 1, 600, 10},
	{1, 0, 0, 600, 2},
}

const midX = 641
const midY = 646

func ClockServer(port int, clock []*Hand) {
	inf, err := os.Open(*clockface)
	if err != nil {
		log.Fatalf("%s: %v", *clockface, err)
	}
	defer inf.Close()
	img, _, err := image.Decode(inf)
	if err != nil {
		log.Fatalf("%s: %v", *clockface, err)
	}
	http.Handle("/clock.jpg", http.HandlerFunc(handler(clock, img)))
	http.Handle("/status", http.HandlerFunc(status(clock)))
	url := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s", url)
	server := &http.Server{Addr: url}
	log.Fatal(server.ListenAndServe())
}

func handler(clock []*Hand, img image.Image) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		c := gg.NewContextForImage(img)
		for i, h := range clock {
			c.SetRGB(handDraw[i].r, handDraw[i].g, handDraw[i].b)
			drawHand(c, h, handDraw[i].length, handDraw[i].width)
		}
		err := jpeg.Encode(w, c.Image(), nil)
		if err != nil {
			log.Printf("Error writing image: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	}
}

func drawHand(c *gg.Context, h *Hand, length, width int) {
	p, r := h.Position()
	p = r - p
	radians := float64(p)*2*math.Pi/float64(r) + math.Pi
	x := float64(length)*math.Sin(radians) + float64(midX)
	y := float64(length)*math.Cos(radians) + float64(midY)
	c.SetLineWidth(float64(width))
	c.DrawLine(float64(midX), float64(midY), x, y)
	c.Stroke()
}

func status(clock []*Hand) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><head>")
		if *refresh != 0 {
			fmt.Fprintf(w, "<meta http-equiv=\"refresh\" content=\"%d\">", *refresh)
		}
		fmt.Fprintf(w, "</head><body><h1>Status</h1>")
		for _, h := range clock {
			fmt.Fprintf(w, "%s: ", h.Name)
			p, r := h.Position()
			fmt.Fprintf(w, "position: %d face size: %d<br>", p, r)
		}
		fmt.Fprintf(w, "<p><a href=\"clock.jpg\">clock face</a><br>")
		fmt.Fprintf(w, "</body>")
	}
}
