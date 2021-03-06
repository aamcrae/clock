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
	"strconv"

	"github.com/fogleman/gg"
)

var clockface = flag.String("clockface", "clock-face.jpg", "Clock face JPEG file")
var refresh = flag.Int("refresh", 10, "Refresh status page number of seconds")

type handDraw struct {
	r      float64
	g      float64
	b      float64
	length int
	width  int
}

var handMap map[string]handDraw = map[string]handDraw{
	"hours":   {0, 0, 1, 400, 30},
	"minutes": {0, 0, 1, 600, 10},
	"seconds": {1, 0, 0, 600, 2},
}

const midX = 641
const midY = 646

// ClockServer starts a HTTP server that displays a clock face and
// status information about the clock.
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
	http.Handle("/adjust", http.HandlerFunc(adjust(clock)))
	url := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s", url)
	server := &http.Server{Addr: url}
	log.Fatal(server.ListenAndServe())
}

// Display the clock face with the current location of the hands drawn upon it.
func handler(clock []*Hand, img image.Image) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		c := gg.NewContextForImage(img)
		for _, h := range clock {
			hd, ok := handMap[h.Name]
			if ok {
				c.SetRGB(hd.r, hd.g, hd.b)
				drawHand(c, h, hd.length, hd.width)
			}
		}
		err := jpeg.Encode(w, c.Image(), nil)
		if err != nil {
			log.Printf("Error writing image: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

	}
}

// Draw a hand onto the image using the requested length and width.
// The positon of the hand is determined from the current physical hand location.
func drawHand(c *gg.Context, h *Hand, length, width int) {
	p, r, _ := h.Get()
	p = r - p
	radians := float64(p)*2*math.Pi/float64(r) + math.Pi
	x := float64(length)*math.Sin(radians) + float64(midX)
	y := float64(length)*math.Cos(radians) + float64(midY)
	c.SetLineWidth(float64(width))
	c.DrawLine(float64(midX), float64(midY), x, y)
	c.Stroke()
}

// status displays the status of each hand of the clock.
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
			p, r, o := h.Get()
			fmt.Fprintf(w, "position: %d offset: %d face size: %d (marks: %d, skipped: %d, fast-forwards %d, adjusted %d)<br>", p, o, r, h.Marks, h.Skipped, h.FastForward, h.Adjusted)
		}
		fmt.Fprintf(w, "<p><a href=\"clock.jpg\">clock face</a><br>")
		fmt.Fprintf(w, "</body>")
	}
}

// adjust applies an adjustment to the hand offset.
// URL parameters are hand=[name] adjust=[value]
func adjust(clock []*Hand) func(http.ResponseWriter, *http.Request) {
	hm := make(map[string]*Hand)
	for _, c := range clock {
		hm[c.Name] = c
	}
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Printf("Adjust request error: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		hand := r.FormValue("hand")
		adj := r.FormValue("adjust")
		h, ok := hm[hand]
		if !ok {
			log.Printf("Unknown hand: %s", hand)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		v, err := strconv.ParseInt(adj, 10, 32)
		if err != nil {
			log.Printf("Illegal value: %s - %v", adj, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><head>")
		fmt.Fprintf(w, "</head><body>")
		p, _, o := h.Get()
		fmt.Fprintf(w, "%s: current offset: %d, position %d<br>", h.Name, o, p)
		h.Adjust(int(v))
		p, _, o = h.Get()
		fmt.Fprintf(w, "%s: new offset: %d, position %d<br>", h.Name, o, p)
		log.Printf("%s: Adjusted offset by %d to %d", h.Name, v, o)
		fmt.Fprintf(w, "</body>")
	}
}
