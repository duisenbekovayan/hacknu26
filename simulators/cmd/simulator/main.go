package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"hacknu/simulators/synth"
)

func main() {
	base := flag.String("url", "http://127.0.0.1:8080", "Базовый URL API")
	interval := flag.Duration("interval", time.Second, "Интервал отправки")
	train := flag.String("train", "LOC-DEMO-001", "Идентификатор поезда")
	seed := flag.Int64("seed", 0, "Seed PRNG (0 — от времени, для повторяемого демо задайте число)")
	flag.Parse()

	var rng *rand.Rand
	if *seed == 0 {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		rng = rand.New(rand.NewSource(*seed))
	}
	g := synth.NewSynthesizer(rng)

	client := &http.Client{Timeout: 8 * time.Second}
	tick := time.NewTicker(*interval)
	defer tick.Stop()

	log.Printf("simulator (synthetic PRNG) -> POST %s/api/v1/telemetry each %s (train=%s)", *base, *interval, *train)

	for range tick.C {
		s := g.NextSample(*train)
		body, err := json.Marshal(s)
		if err != nil {
			log.Println("marshal:", err)
			os.Exit(1)
		}
		req, err := http.NewRequest(http.MethodPost, *base+"/api/v1/telemetry", bytes.NewReader(body))
		if err != nil {
			log.Println("request:", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Println("post:", err)
			continue
		}
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			log.Println("bad status:", resp.Status)
		}
	}
}
